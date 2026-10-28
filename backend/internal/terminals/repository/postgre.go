package terminals

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	terminaldomain "taxi-connect/backend/internal/terminals"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn" // Required for PostgreSQL-specific error handling
)

// Custom error types for the terminals repository
var (
	ErrTerminalNotFound      = errors.New("terminal not found")
	ErrDuplicateTerminalName = errors.New("terminal with this name already exists")
	ErrInvalidLocation       = errors.New("invalid location format")
	ErrUnexpectedNoRows      = errors.New("unexpected no rows returned")
	ErrEmptyName             = errors.New("terminal has no name")
	ErrNameTooLong           = errors.New("name cannot exceed 100 characters")
	ErrInvalidRadius         = errors.New("radius must be positive")
	ErrInvalidCoordinates    = errors.New("invalid coordinates: latitude must be [-90,90], longitude [-180,180]")
)

// postgresTerminalRepository implements TerminalRepository using PostgreSQL/PostGIS
type postgresTerminalRepository struct {
	db *sql.DB
}

// NewPostgresTerminalRepository creates a new PostgreSQL terminal repository
// Returns the interface type so callers depend on abstraction, not implementation
func NewPostgresTerminalRepository(db *sql.DB) terminaldomain.TerminalRepository {
	return &postgresTerminalRepository{db: db}
}

// Create inserts a new terminal and returns its ID
// Note: We rely on the database's DEFAULT NOW() for created_at/updated_at columns
func (r *postgresTerminalRepository) Create(ctx context.Context, terminal terminaldomain.Terminal) (uuid.UUID, error) {
	// Basic validation (could also be done in service layer)
	if terminal.Name == "" {
		return uuid.Nil, ErrEmptyName
	}
	if len(terminal.Name) > 100 {
		return uuid.Nil, ErrNameTooLong
	}

	var id uuid.UUID
	// We only specify name and location - let database handle timestamps via DEFAULT NOW()
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO terminals (name, location)
		 VALUES ($1, ST_GeogFromText($2))
		 RETURNING id`,
		terminal.Name,
		terminal.Location, // Assumes this is already WKT format
	).Scan(&id)

	if err != nil {
		// Handle specific PostgreSQL error types
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return uuid.Nil, ErrUnexpectedNoRows
		case isUniqueViolation(err):
			return uuid.Nil, fmt.Errorf("%w: %s", ErrDuplicateTerminalName, terminal.Name)
		case isInvalidInput(err):
			return uuid.Nil, fmt.Errorf("%w: %v", ErrInvalidLocation, err)
		default:
			return uuid.Nil, fmt.Errorf("failed to create terminal: %w", err)
		}
	}

	return id, nil
}

// GetByID retrieves a terminal by its ID
func (r *postgresTerminalRepository) GetByID(ctx context.Context, id uuid.UUID) (*terminaldomain.Terminal, error) {
	var t terminaldomain.Terminal
	// Note: We scan ALL selected columns in the exact order they appear in the SELECT clause
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, ST_AsText(location), created_at, updated_at
		 FROM terminals WHERE id = $1`,
		id,
	).Scan(&t.ID, &t.Name, &t.Location, &t.CreatedAt, &t.UpdatedAt)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrTerminalNotFound
		default:
			return nil, fmt.Errorf("failed to get terminal: %w", err)
		}
	}

	return &t, nil
}

// GetByName retrieves a terminal by its name
func (r *postgresTerminalRepository) GetByName(ctx context.Context, name string) (*terminaldomain.Terminal, error) {
	var t terminaldomain.Terminal
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, ST_AsText(location), created_at, updated_at
		 FROM terminals WHERE name = $1`,
		name,
	).Scan(&t.ID, &t.Name, &t.Location, &t.CreatedAt, &t.UpdatedAt)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrTerminalNotFound
		default:
			return nil, fmt.Errorf("failed to get terminal by name: %w", err)
		}
	}

	return &t, nil
}

// List returns all terminals ordered by name
func (r *postgresTerminalRepository) List(ctx context.Context) ([]terminaldomain.Terminal, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, ST_AsText(location), created_at, updated_at
		 FROM terminals
		 ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("failed to list terminals: %w", err)
	}
	defer rows.Close() // Important: always close rows when done

	var terminals []terminaldomain.Terminal
	for rows.Next() {
		var t terminaldomain.Terminal
		if err := rows.Scan(&t.ID, &t.Name, &t.Location, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan terminal: %w", err)
		}
		terminals = append(terminals, t)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating terminals: %w", err)
	}

	return terminals, nil
}

// Update modifies an existing terminal
// Note: We explicitly set updated_at = NOW() to update the timestamp
func (r *postgresTerminalRepository) Update(ctx context.Context, terminal terminaldomain.Terminal) error {
	// Basic validation
	if terminal.Name == "" {
		return ErrEmptyName
	}
	if len(terminal.Name) > 100 {
		return ErrNameTooLong
	}

	result, err := r.db.ExecContext(ctx,
		`UPDATE terminals
		 SET name = $1, location = ST_GeogFromText($2), updated_at = NOW()
		 WHERE id = $3`,
		terminal.Name,
		terminal.Location,
		terminal.ID,
	)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return ErrTerminalNotFound
		case isUniqueViolation(err):
			return fmt.Errorf("%w: %s", ErrDuplicateTerminalName, terminal.Name)
		default:
			return fmt.Errorf("failed to update terminal: %w", err)
		}
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrTerminalNotFound
	}

	return nil
}

// Delete removes a terminal by ID
func (r *postgresTerminalRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM terminals WHERE id = $1`,
		id,
	)

	if err != nil {
		return fmt.Errorf("failed to delete terminal: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrTerminalNotFound
	}

	return nil
}

// FindNearby returns terminals within a specified distance (meters) of a point
// Uses ST_DWithin for efficient radius-based search
func (r *postgresTerminalRepository) FindNearby(ctx context.Context, latitude, longitude, radiusMeters float64) ([]terminaldomain.Terminal, error) {	// Validate inputs before hitting the database
	if radiusMeters <= 0 {
		return nil, ErrInvalidRadius
	}
	if latitude < -90 || latitude > 90 || longitude < -180 || longitude > 180 {
		return nil, ErrInvalidCoordinates
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, ST_AsText(location), created_at, updated_at
		 FROM terminals
		 WHERE ST_DWithin(
			 location,
			 ST_MakePoint($1, $2)::geography,
			 $3
		 )
		 ORDER BY name`,
		longitude,    // $1: longitude (first parameter to ST_MakePoint)
		latitude,     // $2: latitude (second parameter to ST_MakePoint)
		radiusMeters, // $3: radius in meters
	)
	if err != nil {
		return nil, fmt.Errorf("failed to find nearby terminals: %w", err)
	}
	defer rows.Close()

	var terminals []terminaldomain.Terminal
	for rows.Next() {
		var t terminaldomain.Terminal
		if err := rows.Scan(&t.ID, &t.Name, &t.Location, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan terminal: %w", err)
		}
		terminals = append(terminals, t)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating nearby terminals: %w", err)
	}

	return terminals, nil
}

// GetRouteEdges returns all directed route edges between terminals.
func (r *postgresTerminalRepository) GetRouteEdges(ctx context.Context) ([]terminaldomain.RouteEdge, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, from_terminal_id, to_terminal_id, travel_time_minutes, created_at, updated_at
		 FROM route_edges
		 ORDER BY from_terminal_id, to_terminal_id`)
	if err != nil {
		return nil, fmt.Errorf("failed to get route edges: %w", err)
	}
	defer rows.Close()

	var edges []terminaldomain.RouteEdge
	for rows.Next() {
		var e terminaldomain.RouteEdge
		if err := rows.Scan(&e.ID, &e.FromTerminalID, &e.ToTerminalID, &e.TravelTimeMinutes, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan route edge: %w", err)
		}
		edges = append(edges, e)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating route edges: %w", err)
	}

	return edges, nil
}

// Helper function to check for PostgreSQL unique violation errors (SQLSTATE 23505)
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// Helper function to check for PostgreSQL input errors (SQLSTATE 22023 or 22007)
func isInvalidInput(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && (pgErr.Code == "22023" || pgErr.Code == "22007")
}
