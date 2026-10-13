package terminals

import (
	"context"

	"github.com/google/uuid"
)

type TerminalRepository interface {
	Create(ctx context.Context, terminal Terminal) (uuid.UUID, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Terminal, error)
	GetByName(ctx context.Context, name string) (*Terminal, error)
	List(ctx context.Context) ([]Terminal, error)
	Update(ctx context.Context, terminal Terminal) error
	Delete(ctx context.Context, id uuid.UUID) error
	FindNearby(ctx context.Context, lat, lon, radiusMeters float64) ([]Terminal, error)
}
