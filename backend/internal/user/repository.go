package user

import (
	"context"

	"github.com/google/uuid"
)

// UserRepository defines the interface for user data access
type UserRepository interface {
	CreateUser(ctx context.Context, user User) (uuid.UUID, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	UpdateUser(ctx context.Context, user User) error
	DeleteUser(ctx context.Context, id uuid.UUID) error
	ListUsers(ctx context.Context, limit int, offset int) ([]User, error)
	GetUserReputation(ctx context.Context, id uuid.UUID) (float64, error)
	IncrementReportCount(ctx context.Context, id uuid.UUID) error
}