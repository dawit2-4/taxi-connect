package terminals

import (
	"context"
	"errors"
	"fmt"

	// taxiConnect "taxi-connect/backend/internal/terminals"
	"github.com/google/uuid"
)

// TerminalService provides business logic for terminal operations
type TerminalService struct {
	repo TerminalRepository
}

// NewTerminalService creates a new TerminalService with the given repository
func NewTerminalService(repo TerminalRepository) *TerminalService {
	return &TerminalService{repo: repo}
}

// Custom error types for the service layer
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

// CreateTerminal validates input and creates a new terminal
func (s *TerminalService) CreateTerminal(ctx context.Context, terminal Terminal) (uuid.UUID, error) {
	// Business logic: Validate terminal name
	if terminal.Name == "" {
		return uuid.Nil, ErrEmptyName
	}
	if len(terminal.Name) > 100 {
		return uuid.Nil, ErrNameTooLong
	}
	// Business logic: Could add more validation here (e.g., name format, reserved words)
	// Example: if strings.Contains(strings.ToUpper(terminal.Name), "TEST") {
	//     return uuid.Nil, errors.New("terminal names cannot contain 'TEST'")
	// }

	// Delegate to repository
	id, err := s.repo.Create(ctx, terminal)
	if err != nil {
		// Add context to repository errors
		switch {
		case errors.Is(err, ErrDuplicateTerminalName):
			return uuid.Nil, fmt.Errorf("failed to create terminal: %w", err)
		case errors.Is(err, ErrInvalidLocation):
			return uuid.Nil, fmt.Errorf("failed to create terminal: %w", err)
		default:
			return uuid.Nil, fmt.Errorf("failed to create terminal: %w", err)
		}
	}

	return id, nil
}

// GetTerminal retrieves a terminal by its ID
func (s *TerminalService) GetTerminal(ctx context.Context, id uuid.UUID) (*Terminal, error) {
	terminal, err := s.repo.GetByID(ctx, id)
	if err != nil {
		// Preserve specific errors like NotFound for HTTP layer to handle
		switch {
		case errors.Is(err, ErrTerminalNotFound):
			return nil, ErrTerminalNotFound
		default:
			return nil, fmt.Errorf("failed to get terminal: %w", err)
		}
	}

	return terminal, nil
}

// GetTerminalByName retrieves a terminal by its name
func (s *TerminalService) GetTerminalByName(ctx context.Context, name string) (*Terminal, error) {
	if name == "" {
		return nil, errors.New("terminal name cannot be empty")
	}

	terminal, err := s.repo.GetByName(ctx, name)
	if err != nil {
		switch {
		case errors.Is(err, ErrTerminalNotFound):
			return nil, ErrTerminalNotFound
		default:
			return nil, fmt.Errorf("failed to get terminal by name: %w", err)
		}
	}

	return terminal, nil
}

// ListTerminals returns all terminals
func (s *TerminalService) ListTerminals(ctx context.Context) ([]Terminal, error) {
	terminals, err := s.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list terminals: %w", err)
	}
	return terminals, nil
}

// UpdateTerminal validates input and updates an existing terminal
func (s *TerminalService) UpdateTerminal(ctx context.Context, terminal Terminal) error {
	// Business logic: Validate terminal
	if terminal.Name == "" {
		return ErrEmptyName
	}
	if len(terminal.Name) > 100 {
		return ErrNameTooLong
	}

	// Delegate to repository
	err := s.repo.Update(ctx, terminal)
	if err != nil {
		switch {
		case errors.Is(err, ErrTerminalNotFound):
			return ErrTerminalNotFound
		case errors.Is(err, ErrDuplicateTerminalName):
			return fmt.Errorf("failed to update terminal: %w", err)
		default:
			return fmt.Errorf("failed to update terminal: %w", err)
		}
	}

	return nil
}

// DeleteTerminal removes a terminal by ID
func (s *TerminalService) DeleteTerminal(ctx context.Context, id uuid.UUID) error {
	err := s.repo.Delete(ctx, id)
	if err != nil {
		switch {
		case errors.Is(err, ErrTerminalNotFound):
			return ErrTerminalNotFound
		default:
			return fmt.Errorf("failed to delete terminal: %w", err)
		}
	}
	return nil
}

// FindTerminalsNear returns terminals within a specified distance (meters) of a point
func (s *TerminalService) FindTerminalsNear(ctx context.Context, latitude, longitude, radiusMeters float64) ([]Terminal, error) {
	// Business logic: Validate inputs (could also be done in repository)
	if radiusMeters <= 0 {
		return nil, ErrInvalidRadius
	}
	if latitude < -90 || latitude > 90 || longitude < -180 || longitude > 180 {
		return nil, ErrInvalidCoordinates
	}

	// Delegate to repository
	terminals, err := s.repo.FindNearby(ctx, latitude, longitude, radiusMeters)
	if err != nil {
		return nil, fmt.Errorf("failed to find nearby terminals: %w", err)
	}
	return terminals, nil
}