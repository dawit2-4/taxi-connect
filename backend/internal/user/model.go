package user

import (
	"time"

	"github.com/google/uuid"
)

// User represents a user in the system.
type User struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	Password  string    `json:"-"` // never JSON encode the password
	CreatedAt time.Time `json:"created_at"`
	IsActive  bool      `json:"is_active"`
	// Reputation fields
	ReputationScore float64 `json:"reputation_score"`
	ReportCount     int     `json:"report_count"`
	// AccuracyRate is the percentage of reports that are considered accurate
	// (to be implemented in future enhancements)
	AccuracyRate float64 `json:"accuracy_rate,omitempty"`
}