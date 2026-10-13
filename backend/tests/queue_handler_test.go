package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"taxi-connect/backend/internal/queue"
	"taxi-connect/backend/internal/terminals"
)

// MockQueueRepositoryHandler mocks the queue repository.
type MockQueueRepositoryHandler struct {
	mock.Mock
}

func (m *MockQueueRepositoryHandler) StoreReport(
	ctx context.Context,
	report queue.QueueReport,
) (uuid.UUID, error) {
	args := m.Called(ctx, report)

	var id uuid.UUID
	if args.Get(0) != nil {
		id = args.Get(0).(uuid.UUID)
	}

	return id, args.Error(1)
}

func (m *MockQueueRepositoryHandler) GetRecentReportsSince(
	ctx context.Context,
	fromID, toID uuid.UUID,
	cutoff time.Time,
) ([]queue.QueueReport, error) {
	args := m.Called(ctx, fromID, toID, cutoff)

	var reports []queue.QueueReport
	if args.Get(0) != nil {
		reports = args.Get(0).([]queue.QueueReport)
	}

	return reports, args.Error(1)
}

// MockTerminalRepositoryHandler mocks the terminal repository.
type MockTerminalRepositoryHandler struct {
	mock.Mock
}

func (m *MockTerminalRepositoryHandler) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (*terminals.Terminal, error) {
	args := m.Called(ctx, id)

	var terminal *terminals.Terminal
	if args.Get(0) != nil {
		terminal = args.Get(0).(*terminals.Terminal)
	}

	return terminal, args.Error(1)
}

func (m *MockTerminalRepositoryHandler) GetByName(
	ctx context.Context,
	name string,
) (*terminals.Terminal, error) {
	panic("not implemented")
}

func (m *MockTerminalRepositoryHandler) List(
	ctx context.Context,
) ([]terminals.Terminal, error) {
	panic("not implemented")
}

func (m *MockTerminalRepositoryHandler) Update(
	ctx context.Context,
	terminal terminals.Terminal,
) error {
	panic("not implemented")
}

func (m *MockTerminalRepositoryHandler) Delete(
	ctx context.Context,
	id uuid.UUID,
) error {
	panic("not implemented")
}

func (m *MockTerminalRepositoryHandler) FindNearby(
	ctx context.Context,
	latitude, longitude, radiusMeters float64,
) ([]terminals.Terminal, error) {
	panic("not implemented")
}

func (m *MockTerminalRepositoryHandler) Create(
	ctx context.Context,
	terminal terminals.Terminal,
) (uuid.UUID, error) {
	args := m.Called(ctx, terminal)

	var id uuid.UUID
	if args.Get(0) != nil {
		id = args.Get(0).(uuid.UUID)
	}

	return id, args.Error(1)
}

// Compile-time interface check.
var _ terminals.TerminalRepository = (*MockTerminalRepositoryHandler)(nil)

// TestStoreReportHandler tests the HTTP handler for submitting queue reports.
func TestStoreReportHandler(t *testing.T) {
	fromID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	toID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	reportID := uuid.MustParse("12345678-1234-1234-1234-123456789012")

	tests := []struct {
		name         string
		body         []byte
		setupFn      func(*MockQueueRepositoryHandler, *MockTerminalRepositoryHandler)
		expectedCode int
		expectedBody map[string]interface{}
	}{
		{
			name: "valid report",
			body: []byte(fmt.Sprintf(`{
				"from_terminal_id": "%s",
				"to_terminal_id": "%s",
				"reported_wait_minutes": 12,
				"reported_queue_length": 5,
				"reporter_notes": "Test note"
			}`, fromID, toID)),
			setupFn: func(
				queueRepo *MockQueueRepositoryHandler,
				termRepo *MockTerminalRepositoryHandler,
			) {
				termRepo.On("GetByID", mock.Anything, fromID).
					Return(&terminals.Terminal{ID: fromID, Name: "Origin"}, nil).Once()

				termRepo.On("GetByID", mock.Anything, toID).
					Return(&terminals.Terminal{ID: toID, Name: "Destination"}, nil).Once()

				queueRepo.On(
					"StoreReport",
					mock.Anything,
					mock.AnythingOfType("queue.QueueReport"),
				).Return(reportID, nil).Once()
			},
			expectedCode: http.StatusCreated,
			expectedBody: map[string]interface{}{
				"id": reportID.String(),
			},
		},
		{
			name:         "invalid JSON",
			body:         []byte(`{invalid json}`),
			setupFn:      func(_ *MockQueueRepositoryHandler, _ *MockTerminalRepositoryHandler) {},
			expectedCode: http.StatusBadRequest,
		},
		{
			name: "missing from terminal",
			body: []byte(`{
				"to_terminal_id": "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
				"reported_wait_minutes": 10
			}`),
			setupFn:      func(_ *MockQueueRepositoryHandler, _ *MockTerminalRepositoryHandler) {},
			expectedCode: http.StatusBadRequest,
		},
		{
			name: "invalid UUID format",
			body: []byte(`{
				"from_terminal_id": "not-a-uuid",
				"to_terminal_id": "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
				"reported_wait_minutes": 10
			}`),
			setupFn:      func(_ *MockQueueRepositoryHandler, _ *MockTerminalRepositoryHandler) {},
			expectedCode: http.StatusBadRequest,
		},
		{
			name: "service validation error",
			body: []byte(fmt.Sprintf(`{
        "from_terminal_id": "%s",
        "to_terminal_id": "%s",
        "reported_wait_minutes": -5
    }`, fromID, toID)),
			setupFn: func(
				_ *MockQueueRepositoryHandler,
				_ *MockTerminalRepositoryHandler,
			) {
				// No repository calls should happen for invalid input.
			},
			expectedCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Fresh mocks for every subtest.
			mockQueueRepo := new(MockQueueRepositoryHandler)
			mockTermRepo := new(MockTerminalRepositoryHandler)

			service := queue.NewQueueService(mockQueueRepo, mockTermRepo)
			handler := queue.NewQueueHandler(service)

			if tt.setupFn != nil {
				tt.setupFn(mockQueueRepo, mockTermRepo)
			}

			req := httptest.NewRequest(
				http.MethodPost,
				"/queue/report",
				bytes.NewReader(tt.body),
			)
			req.Header.Set("Content-Type", "application/json")

			recorder := httptest.NewRecorder()
			handler.StoreReportHandler(recorder, req)

			assert.Equal(t, tt.expectedCode, recorder.Code)

			if tt.expectedBody != nil {
				var actualBody map[string]interface{}
				err := json.NewDecoder(recorder.Body).Decode(&actualBody)
				assert.NoError(t, err)

				for key, expected := range tt.expectedBody {
					assert.Equal(t, expected, actualBody[key])
				}
			}

			mockQueueRepo.AssertExpectations(t)
			mockTermRepo.AssertExpectations(t)
		})
	}
}
