package queue

import (
	"time"

	"github.com/google/uuid"
)

// QueueReport represents a user's queue observation.
type QueueReport struct {
	ID           uuid.UUID `json:"id"`
	FromTerminalID uuid.UUID `json:"from_terminal_id"`
	ToTerminalID   uuid.UUID `json:"to_terminal_id"`
	ReportedQueueLength *int    `json:"reported_queue_length,omitempty"`
	ReportedWaitMinutes *int    `json:"reported_wait_minutes,omitempty"`
	ReporterNotes   string    `json:"reporter_notes,omitempty"`
	ReportedAt      time.Time `json:"reported_at"`
	// UserID is the ID of the user who submitted this report (nullable for reports submitted before auth)
	UserID          *uuid.UUID `json:"user_id,omitempty"`
}

// QueueEstimate represents the estimated wait time for a route.
type QueueEstimate struct {
	FromTerminalID uuid.UUID `json:"from_terminal_id"`
	ToTerminalID   uuid.UUID `json:"to_terminal_id"`
	CurrentWaitTime int       `json:"current_wait_time"`
	ConfidenceScore float32   `json:"confidence_score"`
	ReportedCount   int       `json:"reported_count"`
	LastUpdated     time.Time `json:"last_updated"`
}