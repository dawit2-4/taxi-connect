package queue

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

// postgresQueueRepository implements QueueRepository using PostgreSQL
type postgresQueueRepository struct {
	db *sql.DB
}


var (
	ErrUnexpectedNoRows = errors.New("unexpected no rows returned")
	ErrDuplicateReport  = errors.New("this report already exists")
)


// NewPostgresQueueRepository creates a new PostgreSQL queue repository
// Returns the interface type so callers depend on abstraction, not implementation
func NewPostgresQueueRepository(db *sql.DB) QueueRepository {
	return &postgresQueueRepository{db: db}
}

// StoreReport saves a validated user observation
func (r *postgresQueueRepository) StoreReport(ctx context.Context, report QueueReport) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO queue_reports (from_terminal_id, to_terminal_id, reported_wait_minutes, reported_queue_length, reporter_notes, reported_at)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id`,
		report.FromTerminalID, report.ToTerminalID, report.ReportedWaitMinutes, report.ReportedQueueLength, report.ReporterNotes, report.ReportedAt,
	).Scan(&id)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return uuid.Nil, fmt.Errorf("failed to store queue report: %w", ErrUnexpectedNoRows)
		default:
			return uuid.Nil, fmt.Errorf("failed to store queue report: %w", err)
		}
	}

	return id, nil
}

// GetRecentReportsSince returns raw observations for a specific route
// that occurred after the given cutoff time.
func (r *postgresQueueRepository) GetRecentReportsSince(ctx context.Context, fromID, toID uuid.UUID, cutoff time.Time) ([]QueueReport, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, from_terminal_id, to_terminal_id, reported_wait_minutes, reported_queue_length, reporter_notes, reported_at
		 FROM queue_reports
		 WHERE from_terminal_id = $1 AND to_terminal_id = $2 AND reported_at >= $3
		 ORDER BY reported_at DESC`,
		fromID, toID, cutoff,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent queue reports: %w", err)
	}
	defer rows.Close()

	var reports []QueueReport
	for rows.Next() {
		var report QueueReport
		if err := rows.Scan(&report.ID, &report.FromTerminalID, &report.ToTerminalID, &report.ReportedWaitMinutes, &report.ReportedQueueLength, &report.ReporterNotes, &report.ReportedAt); err != nil {
			return nil, fmt.Errorf("failed to scan queue report: %w", err)
		}
		reports = append(reports, report)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating queue reports: %w", err)
	}

	return reports, nil
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