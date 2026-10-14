package user

import (
	"context"
	"database/sql"
	"errors"

	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

// postgresUserRepository implements UserRepository using PostgreSQL
type postgresUserRepository struct {
	db *sql.DB
}

// NewPostgresUserRepository creates a new PostgreSQL user repository
// Returns the interface type so callers depend on abstraction, not implementation
func NewPostgresUserRepository(db *sql.DB) UserRepository {
	return &postgresUserRepository{db: db}
}

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user with this email already exists")
	ErrUnexpectedNoRows  = errors.New("unexpected no rows returned")
)

func (r *postgresUserRepository) CreateUser(ctx context.Context, user User) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.QueryRowContext(ctx, `INSERT INTO users (email, password, created_at, is_active, reputation_score, report_count, accuracy_rate)
																			VALUES ($1, $2, $3, $4, $5, $6, $7)
																			RETURNING id`,
																		user.Email, user.Password, user.CreatedAt, user.IsActive, user.ReputationScore, user.ReportCount, user.AccuracyRate).Scan(&id)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return uuid.Nil, fmt.Errorf("failed to create user: %w", ErrUnexpectedNoRows)
		default:
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				return uuid.Nil, fmt.Errorf("failed to create user: %w", ErrUserAlreadyExists)
			}

			return uuid.Nil, fmt.Errorf("failed to create user: %w", err)
		}
	}

	return id, nil
}

func (r *postgresUserRepository) GetUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	var user User
	err := r.db.QueryRowContext(ctx, `SELECT id, email, password, is_active, reputation_score, report_count, accuracy_rate
																			FROM users
																			WHERE id = $1`,
																		id,
	).Scan(&user.ID, &user.Email, &user.Password, &user.CreatedAt, &user.IsActive, &user.ReputationScore, user.ReportCount, user.AccuracyRate)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrUserNotFound
		default:
			return nil, fmt.Errorf("failed to get user by ID: %w", err)
		}
	}

	return &user, nil
}

func (r *postgresUserRepository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	var user User
	err := r.db.QueryRowContext(ctx, `SELECT id, email, password, is_active, reputation_score, report_count, accuracy_rate
																			FROM users
																			WHERE email = $1`,
																		email,
	).Scan(&user.ID, &user.Email, &user.Password, &user.CreatedAt, &user.IsActive, &user.ReputationScore, user.ReportCount, user.AccuracyRate)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrUserNotFound
		default:
			return nil, fmt.Errorf("failed to get user by email: %w", err)
		}
	}

	return &user, nil
}

// UpdateUser updates an existing user's fields.
func (r *postgresUserRepository) UpdateUser(ctx context.Context, user User) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users
		 SET email = $2, password = $3, created_at = $4, is_active = $5, reputation_score = $6, report_count = $7, accuracy_rate
		 WHERE id = $1`,
		user.ID, user.Email, user.Password, user.CreatedAt, user.IsActive, user.ReputationScore, user.ReportCount, user.AccuracyRate,
	)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	return nil
}

// DeleteUser removes a user by ID.
func (r *postgresUserRepository) DeleteUser(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM users WHERE id = $1`,
		id,
	)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}

// ListUsers returns a paginated list of users (for admin/leaderboard).
func (r *postgresUserRepository) ListUsers(ctx context.Context, limit int, offset int) ([]User, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, email, password, created_at, is_active, reputation_score, report_count, accuracy_rate
		 FROM users
		 ORDER BY created_at DESC
		 LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.Email, &user.Password, &user.CreatedAt, &user.IsActive, &user.ReputationScore, user.ReportCount, user.AccuracyRate); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating users: %w", err)
	}

	return users, nil
}

// GetUserReputation returns just the reputation score for a user.
func (r *postgresUserRepository) GetUserReputation(ctx context.Context, id uuid.UUID) (float64, error) {
	var reputation float64
	err := r.db.QueryRowContext(ctx,
		`SELECT reputation_score FROM users WHERE id = $1`,
		id,
	).Scan(&reputation)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrUserNotFound
		}
		return 0, fmt.Errorf("failed to get user reputation: %w", err)
	}

	return reputation, nil
}

// IncrementReportCount increases a user's report count by 1.
func (r *postgresUserRepository) IncrementReportCount(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET report_count = report_count + 1 WHERE id = $1`,
		id,
	)
	if err != nil {
		return fmt.Errorf("failed to increment report count: %w", err)
	}
	return nil
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