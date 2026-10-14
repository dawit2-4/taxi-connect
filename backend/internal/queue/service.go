package queue

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	terminalrepo "taxi-connect/backend/internal/terminals"

	"github.com/google/uuid"
)

// UserRepository defines the interface for user data access.
// This is defined here to avoid import cycles with the user package.
type UserRepository interface {
	IncrementReportCount(ctx context.Context, id uuid.UUID) error
}

// QueueService provides business logic for queue operations
type QueueService struct {
	repo     QueueRepository
	userRepo UserRepository
	termRepo terminalrepo.TerminalRepository
	notifier QueueNotifier
}

type QueueNotifier interface {
	NotifyQueueUpdate(fromID, toID uuid.UUID)
}

// NewQueueService creates a new QueueService with the given repositories
func NewQueueService(repo QueueRepository, userRepo UserRepository, termRepo terminalrepo.TerminalRepository, notifier QueueNotifier) *QueueService {
	return &QueueService{repo: repo, userRepo: userRepo, termRepo: termRepo, notifier: notifier}
}

// Custom error types for the service layer
var (
	ErrNoRecentData     = errors.New("no recent wait-time data available")
	ErrInvalidReport    = errors.New("invalid queue report")
	ErrSameTerminals    = errors.New("origin and destination terminals cannot be the same")
	ErrTerminalNotFound = errors.New("terminal not found")
)

func (s *QueueService) notifyQueueUpdate(fromID, toID uuid.UUID) {
	if s.notifier != nil {
		s.notifier.NotifyQueueUpdate(fromID, toID)
	}
}

// StoreReport validates and stores a user's queue observation
// Implements MVP decision #1: Validate terminals exist, require at least one of wait time/queue length,
// store both if provided, reject negative values and identical terminals
// Associates report with authenticated user if available in context

func (s *QueueService) StoreReport(
	ctx context.Context,
	report QueueReport,
) (uuid.UUID, error) {
	// Extract user ID from context if available (set by JWT middleware)
	if userIDVal := ctx.Value("userID"); userIDVal != nil {
		if userID, ok := userIDVal.(uuid.UUID); ok {
			report.UserID = &userID
		}
	}

	// Validate required terminal IDs.
	if report.FromTerminalID == uuid.Nil || report.ToTerminalID == uuid.Nil {
		return uuid.Nil, ErrInvalidReport
	}

	// Reject identical terminals before making repository calls.
	if report.FromTerminalID == report.ToTerminalID {
		return uuid.Nil, ErrSameTerminals
	}

	// Require at least one of wait time or queue length.
	if report.ReportedWaitMinutes == nil &&
		report.ReportedQueueLength == nil {
		return uuid.Nil, ErrInvalidReport
	}

	// Reject negative values.
	if report.ReportedWaitMinutes != nil &&
		*report.ReportedWaitMinutes < 0 {
		return uuid.Nil, ErrInvalidReport
	}

	if report.ReportedQueueLength != nil &&
		*report.ReportedQueueLength < 0 {
		return uuid.Nil, ErrInvalidReport
	}

	// Validate the origin terminal exists.
	_, err := s.termRepo.GetByID(ctx, report.FromTerminalID)
	if err != nil {
		if errors.Is(err, terminalrepo.ErrTerminalNotFound) {
			return uuid.Nil, fmt.Errorf(
				"origin terminal not found: %w",
				ErrTerminalNotFound,
			)
		}
		return uuid.Nil, fmt.Errorf(
			"failed to validate origin terminal: %w",
			err,
		)
	}

	// Validate the destination terminal exists.
	_, err = s.termRepo.GetByID(ctx, report.ToTerminalID)
	if err != nil {
		if errors.Is(err, terminalrepo.ErrTerminalNotFound) {
			return uuid.Nil, fmt.Errorf(
				"destination terminal not found: %w",
				ErrTerminalNotFound,
			)
		}
		return uuid.Nil, fmt.Errorf(
			"failed to validate destination terminal: %w",
			err,
		)
	}

	// Store the report only after validation succeeds.
	id, err := s.repo.StoreReport(ctx, report)
	if err != nil {
		return uuid.Nil, err
	}

	// Increment the user's report count if user ID is available
	if report.UserID != nil {
		if err := s.userRepo.IncrementReportCount(ctx, *report.UserID); err != nil {
			// Log error but don't fail the request - report was still stored successfully
			// In production, you might want to use a proper logger
			fmt.Printf("Warning: failed to increment report count for user %s: %v\n", report.UserID.String(), err)
		}
	}

	// 👇 WEBSOCKET NOTIFICATION: Notify subscribers of updated data
	// This is the ONLY change needed for WebSocket integration
	s.notifyQueueUpdate(report.FromTerminalID, report.ToTerminalID)

	return id, nil
}

// GetEstimate calculates wait time estimate for a route
// Implements MVP decision #2: Consider reports from last 10 minutes, recency-weighted average of wait-time reports,
// queue-length-only as supplementary, explicit no-estimate when no data
// Implements MVP decision #3: Simple low/medium/high confidence
func (s *QueueService) GetEstimate(ctx context.Context, fromID, toID uuid.UUID) (*QueueEstimate, error) {
	// Consider reports from last 10 minutes
	cutoff := time.Now().Add(-10 * time.Minute)

	// Get recent reports from repository
	reports, err := s.repo.GetRecentReportsSince(ctx, fromID, toID, cutoff)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent reports: %w", err)
	}

	// If no reports at all, return explicit no-estimate error
	if len(reports) == 0 {
		return nil, ErrNoRecentData
	}

	// Filter reports to only those with non-nil ReportedWaitMinutes (for wait time estimation)
	var waitReports []QueueReport
	for _, report := range reports {
		if report.ReportedWaitMinutes != nil {
			waitReports = append(waitReports, report)
		}
	}

	// If no wait-time reports, return explicit no-estimate error
	if len(waitReports) == 0 {
		return nil, ErrNoRecentData
	}

	// Apply recency-weighted average algorithm
	// Weight = exponential decay: newer reports have more influence
	// weight = exp(-age_minutes / tau) where tau = 5 minutes (adjustable parameter)
	const tau = 5.0 // time constant for exponential decay (minutes)

	var sumWeightedWait float64
	var sumWeights float64
	var mostRecent time.Time

	for _, report := range waitReports {
		age := time.Since(report.ReportedAt).Minutes()
		// Exponential decay weight: newer reports get higher weight
		weight := float64(math.Exp(-age / tau))
		sumWeightedWait += float64(*report.ReportedWaitMinutes) * weight
		sumWeights += weight

		// Track the most recent report for LastUpdated field
		if report.ReportedAt.After(mostRecent) {
			mostRecent = report.ReportedAt
		}
	}

	// Calculate estimated wait time
	var estimatedWait int
	if sumWeights > 0 {
		estimatedWait = int(sumWeightedWait / sumWeights)
	} else {
		// Fallback: simple average if weights somehow sum to zero
		sum := 0
		count := 0
		for _, report := range waitReports {
			sum += *report.ReportedWaitMinutes
			count++
		}
		if count > 0 {
			estimatedWait = sum / count
		}
	}

	// Calculate confidence score (low/medium/high based on report count and freshness)
	var confidenceScore float32
	reportCount := len(waitReports)

	// Base confidence on report count and age of reports
	if reportCount >= 4 {
		// Check if all reports are relatively fresh (within 5 minutes)
		allRecent := true
		for _, report := range waitReports {
			if time.Since(report.ReportedAt).Minutes() > 5 {
				allRecent = false
				break
			}
		}
		if allRecent {
			confidenceScore = 0.9 // High confidence
		} else {
			confidenceScore = 0.7 // Medium-High confidence
		}
	} else if reportCount >= 2 {
		// Check if reports are reasonably fresh (within 8 minutes)
		reasonablyFresh := true
		for _, report := range waitReports {
			if time.Since(report.ReportedAt).Minutes() > 8 {
				reasonablyFresh = false
				break
			}
		}
		if reasonablyFresh {
			confidenceScore = 0.6 // Medium confidence
		} else {
			confidenceScore = 0.4 // Low-Medium confidence
		}
	} else {
		// Single report or no reports (shouldn't happen due to earlier check, but safe)
		confidenceScore = 0.2 // Low confidence
	}

	// Ensure confidence is in [0, 1] range
	if confidenceScore < 0 {
		confidenceScore = 0
	} else if confidenceScore > 1 {
		confidenceScore = 1
	}

	// Return the estimate
	return &QueueEstimate{
		FromTerminalID:  fromID,
		ToTerminalID:    toID,
		CurrentWaitTime: estimatedWait,
		ConfidenceScore: confidenceScore,
		ReportedCount:   reportCount,
		LastUpdated:     mostRecent,
	}, nil
}
