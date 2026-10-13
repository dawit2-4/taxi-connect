package queue

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// QueueRepository defines the interface for queue data access
type QueueRepository interface {
	StoreReport(ctx context.Context, report QueueReport) (uuid.UUID, error)
	GetRecentReportsSince(ctx context.Context, fromID, toID uuid.UUID, cutoff time.Time) ([]QueueReport, error)
}