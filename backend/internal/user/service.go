package user

import (
	"context"
	"fmt"
	"time"

	"taxi-connect/backend/internal/auth"

	"github.com/google/uuid"
)

type UserService struct {
	repo UserRepository
}

func NewUserService(repo UserRepository) *UserService {
	return &UserService{
		repo: repo,
	}
}

// RegisterUser creates a new user with a hashed password.
func (s *UserService) RegisterUser(
	ctx context.Context,
	email, password string,
) (uuid.UUID, error) {
	// 1. Hash the password.
	hashedPassword, err := auth.HashPassword(password)
	if err != nil {
		return uuid.Nil, fmt.Errorf("hash password: %w", err)
	}

	// 2. Create the user.
	user := User{
		Email:           email,
		Password:        hashedPassword,
		CreatedAt:       time.Now(),
		IsActive:        true,
		ReputationScore: 0.0,
		ReportCount:     0,
		AccuracyRate:    0.0,
	}

	// 3. Store the user.
	id, err := s.repo.CreateUser(ctx, user)
	if err != nil {
		return uuid.Nil, fmt.Errorf("create user: %w", err)
	}

	// 4. Return the generated user ID.
	return id, nil
}

// UpdateReputation updates a user's reputation score.
func (s *UserService) UpdateReputation(
	ctx context.Context,
	id uuid.UUID,
	score float64,
) error {
	// Validate the score.
	if score < 0.0 || score > 1.0 {
		return fmt.Errorf("reputation score must be between 0 and 1")
	}

	// Retrieve the user.
	user, err := s.repo.GetUserByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}

	// Update the reputation score.
	user.ReputationScore = score

	// Persist the changes.
	if err := s.repo.UpdateUser(ctx, *user); err != nil {
		return fmt.Errorf("update user reputation: %w", err)
	}

	return nil
}

// AuthenticateUser verifies a user's email and password.
func (s *UserService) AuthenticateUser(
	ctx context.Context,
	email, password string,
) (uuid.UUID, error) {
	// Retrieve the user by email.
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid credentials")
	}

	// Reject inactive accounts.
	if !user.IsActive {
		return uuid.Nil, fmt.Errorf("invalid credentials")
	}

	// Verify the password.
	if !auth.CheckPassword(user.Password, password) {
		return uuid.Nil, fmt.Errorf("invalid credentials")
	}

	return user.ID, nil
}

// GetUserByID retrieves a user by their UUID.
func (s *UserService) GetUserByID(
	ctx context.Context,
	id uuid.UUID,
) (*User, error) {
	user, err := s.repo.GetUserByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get user by ID: %w", err)
	}

	return user, nil
}

// IncrementReportCount increments a user's report count.
func (s *UserService) IncrementReportCount(
	ctx context.Context,
	id uuid.UUID,
) error {
	if err := s.repo.IncrementReportCount(ctx, id); err != nil {
		return fmt.Errorf("increment report count: %w", err)
	}

	return nil
}
