package tests

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"taxi-connect/backend/internal/queue"
	terminalrepo "taxi-connect/backend/internal/terminals"
)

type MockQueueRepository struct {
	mock.Mock
}

func (m *MockQueueRepository) StoreReport(
	ctx context.Context,
	report queue.QueueReport,
) (uuid.UUID, error) {
	args := m.Called(ctx, report)

	var id uuid.UUID
	if args.Get(0) != nil {
		id, _ = args.Get(0).(uuid.UUID)
	}
	return id, mockError(args, 1)
}

func (m *MockQueueRepository) GetRecentReportsSince(
	ctx context.Context,
	fromID, toID uuid.UUID,
	cutoff time.Time,
) ([]queue.QueueReport, error) {
	args := m.Called(ctx, fromID, toID, cutoff)

	var reports []queue.QueueReport
	if args.Get(0) != nil {
		reports, _ = args.Get(0).([]queue.QueueReport)
	}
	return reports, mockError(args, 1)
}

type MockTerminalRepository struct {
	mock.Mock
}

func (m *MockTerminalRepository) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (*terminalrepo.Terminal, error) {
	args := m.Called(ctx, id)

	var terminal *terminalrepo.Terminal
	if args.Get(0) != nil {
		terminal, _ = args.Get(0).(*terminalrepo.Terminal)
	}
	return terminal, mockError(args, 1)
}

func (m *MockTerminalRepository) GetByName(
	ctx context.Context,
	name string,
) (*terminalrepo.Terminal, error) {
	panic("not implemented")
}

func (m *MockTerminalRepository) List(
	ctx context.Context,
) ([]terminalrepo.Terminal, error) {
	panic("not implemented")
}

func (m *MockTerminalRepository) Update(
	ctx context.Context,
	terminal terminalrepo.Terminal,
) error {
	panic("not implemented")
}

func (m *MockTerminalRepository) Delete(
	ctx context.Context,
	id uuid.UUID,
) error {
	panic("not implemented")
}

func (m *MockTerminalRepository) FindNearby(
	ctx context.Context,
	latitude, longitude, radiusMeters float64,
) ([]terminalrepo.Terminal, error) {
	panic("not implemented")
}

func (m *MockTerminalRepository) Create(
	ctx context.Context,
	terminal terminalrepo.Terminal,
) (uuid.UUID, error) {
	args := m.Called(ctx, terminal)

	var id uuid.UUID
	if args.Get(0) != nil {
		id, _ = args.Get(0).(uuid.UUID)
	}
	return id, mockError(args, 1)
}

func mockError(args mock.Arguments, index int) error {
	if args.Get(index) == nil {
		return nil
	}
	err, _ := args.Get(index).(error)
	return err
}

func ptrInt(v int) *int {
	return &v
}

func TestServiceReportValidations(t *testing.T) {
	ctx := context.Background()
	fromID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	toID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")

	tests := []struct {
		name          string
		report        queue.QueueReport
		setup         func(*MockTerminalRepository, *MockQueueRepository)
		expectedError error
	}{
		{
			name: "missing origin ID",
			report: queue.QueueReport{
				ToTerminalID:        toID,
				ReportedWaitMinutes: ptrInt(10),
			},
			expectedError: queue.ErrInvalidReport,
		},
		{
			name: "missing destination ID",
			report: queue.QueueReport{
				FromTerminalID:      fromID,
				ReportedWaitMinutes: ptrInt(10),
			},
			expectedError: queue.ErrInvalidReport,
		},
		{
			name: "origin terminal not found",
			report: queue.QueueReport{
				FromTerminalID:      fromID,
				ToTerminalID:        toID,
				ReportedWaitMinutes: ptrInt(10),
			},
			setup: func(tr *MockTerminalRepository, _ *MockQueueRepository) {
				tr.On("GetByID", mock.Anything, fromID).
					Return(nil, terminalrepo.ErrTerminalNotFound).Once()
			},
			expectedError: queue.ErrTerminalNotFound,
		},
		{
			name: "destination terminal not found",
			report: queue.QueueReport{
				FromTerminalID:      fromID,
				ToTerminalID:        toID,
				ReportedWaitMinutes: ptrInt(10),
			},
			setup: func(tr *MockTerminalRepository, _ *MockQueueRepository) {
				tr.On("GetByID", mock.Anything, fromID).
					Return(&terminalrepo.Terminal{ID: fromID, Name: "Origin"}, nil).Once()
				tr.On("GetByID", mock.Anything, toID).
					Return(nil, terminalrepo.ErrTerminalNotFound).Once()
			},
			expectedError: queue.ErrTerminalNotFound,
		},
		{
			name: "same terminals",
			report: queue.QueueReport{
				FromTerminalID:      fromID,
				ToTerminalID:        fromID,
				ReportedWaitMinutes: ptrInt(10),
			},
			// Same-terminal validation happens before repository calls.
			expectedError: queue.ErrSameTerminals,
		},
		{
			name: "missing wait time and queue length",
			report: queue.QueueReport{
				FromTerminalID: fromID,
				ToTerminalID:   toID,
			},
			// Invalid report validation happens before repository calls.
			expectedError: queue.ErrInvalidReport,
		},
		{
			name: "negative wait time",
			report: queue.QueueReport{
				FromTerminalID:      fromID,
				ToTerminalID:        toID,
				ReportedWaitMinutes: ptrInt(-5),
			},
			expectedError: queue.ErrInvalidReport,
		},
		{
			name: "negative queue length",
			report: queue.QueueReport{
				FromTerminalID:      fromID,
				ToTerminalID:        toID,
				ReportedQueueLength: ptrInt(-3),
			},
			expectedError: queue.ErrInvalidReport,
		},
		{
			name: "valid report",
			report: queue.QueueReport{
				FromTerminalID:      fromID,
				ToTerminalID:        toID,
				ReportedWaitMinutes: ptrInt(12),
				ReportedQueueLength: ptrInt(5),
				ReporterNotes:       "Test note",
				ReportedAt:          time.Now(),
			},
			setup: func(tr *MockTerminalRepository, qr *MockQueueRepository) {
				tr.On("GetByID", mock.Anything, fromID).
					Return(&terminalrepo.Terminal{ID: fromID, Name: "Origin"}, nil).Once()
				tr.On("GetByID", mock.Anything, toID).
					Return(&terminalrepo.Terminal{ID: toID, Name: "Destination"}, nil).Once()
				qr.On("StoreReport", mock.Anything, mock.Anything).
					Return(fromID, nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := new(MockTerminalRepository)
			qr := new(MockQueueRepository)
			service := queue.NewQueueService(qr, tr)

			if tt.setup != nil {
				tt.setup(tr, qr)
			}

			id, err := service.StoreReport(ctx, tt.report)

			if tt.expectedError != nil {
				assert.ErrorIs(t, err, tt.expectedError)
				assert.Equal(t, uuid.Nil, id)
			} else {
				assert.NoError(t, err)
				assert.NotEqual(t, uuid.Nil, id)
			}

			tr.AssertExpectations(t)
			qr.AssertExpectations(t)
			if tt.expectedError == queue.ErrInvalidReport ||
				tt.expectedError == queue.ErrSameTerminals {
				tr.AssertNotCalled(t, "GetByID", mock.Anything, mock.Anything)
				qr.AssertNotCalled(t, "StoreReport", mock.Anything, mock.Anything)
			}
		})
	}
}

func TestServiceGetEstimate(t *testing.T) {
	fromID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	toID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")

	tests := []struct {
		name          string
		reports       []queue.QueueReport
		expectedError error
		expectedWait  int
		expectedScore float32
		expectedCount int
	}{
		{
			name:          "no reports",
			reports:       []queue.QueueReport{},
			expectedError: queue.ErrNoRecentData,
		},
		{
			name: "reports without wait times",
			reports: []queue.QueueReport{
				{
					FromTerminalID:      fromID,
					ToTerminalID:        toID,
					ReportedQueueLength: ptrInt(5),
					ReportedAt:          time.Now().Add(-2 * time.Minute),
				},
			},
			expectedError: queue.ErrNoRecentData,
		},
		{
			name: "single wait time report",
			reports: []queue.QueueReport{
				{
					FromTerminalID:      fromID,
					ToTerminalID:        toID,
					ReportedWaitMinutes: ptrInt(10),
					ReportedAt:          time.Now().Add(-2 * time.Minute),
				},
			},
			expectedWait:  10,
			expectedScore: 0.2,
			expectedCount: 1,
		},
		{
			name: "multiple recent reports with recency weighting",
			reports: []queue.QueueReport{
				{
					FromTerminalID:      fromID,
					ToTerminalID:        toID,
					ReportedWaitMinutes: ptrInt(8),
					ReportedAt:          time.Now().Add(-1 * time.Minute),
				},
				{
					FromTerminalID:      fromID,
					ToTerminalID:        toID,
					ReportedWaitMinutes: ptrInt(12),
					ReportedAt:          time.Now().Add(-3 * time.Minute),
				},
				{
					FromTerminalID:      fromID,
					ToTerminalID:        toID,
					ReportedWaitMinutes: ptrInt(15),
					ReportedAt:          time.Now().Add(-5 * time.Minute),
				},
			},
			expectedWait:  10,
			expectedScore: 0.6,
			expectedCount: 3,
		},
		{
			name: "queue-length-only report does not affect wait estimate",
			reports: []queue.QueueReport{
				{
					FromTerminalID:      fromID,
					ToTerminalID:        toID,
					ReportedWaitMinutes: ptrInt(14),
					ReportedAt:          time.Now().Add(-3 * time.Minute),
				},
				{
					FromTerminalID:      fromID,
					ToTerminalID:        toID,
					ReportedQueueLength: ptrInt(7),
					ReportedAt:          time.Now().Add(-1 * time.Minute),
				},
			},
			expectedWait:  14,
			expectedScore: 0.2,
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qr := new(MockQueueRepository)
			tr := new(MockTerminalRepository)
			service := queue.NewQueueService(qr, tr)

			qr.On(
				"GetRecentReportsSince",
				mock.Anything,
				fromID,
				toID,
				mock.Anything,
			).Return(tt.reports, nil).Once()

			estimate, err := service.GetEstimate(context.Background(), fromID, toID)

			if tt.expectedError != nil {
				assert.ErrorIs(t, err, tt.expectedError)
				assert.Nil(t, estimate)
			} else {
				assert.NoError(t, err)
				if assert.NotNil(t, estimate) {
					assert.Equal(t, fromID, estimate.FromTerminalID)
					assert.Equal(t, toID, estimate.ToTerminalID)
					assert.Equal(t, tt.expectedWait, estimate.CurrentWaitTime)
					assert.InDelta(t, tt.expectedScore, estimate.ConfidenceScore, 0.01)
					assert.Equal(t, tt.expectedCount, estimate.ReportedCount)
				}
			}

			qr.AssertExpectations(t)
		})
	}
}
