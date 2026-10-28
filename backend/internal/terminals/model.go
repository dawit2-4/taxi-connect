package terminals

import (
	"time"

	"github.com/google/uuid"
)

type Terminal struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Location  string    `json:"location"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// RouteEdge represents a directed route between two terminals with travel time.
type RouteEdge struct {
	ID                uuid.UUID `json:"id"`
	FromTerminalID    uuid.UUID `json:"from_terminal_id"`
	ToTerminalID      uuid.UUID `json:"to_terminal_id"`
	TravelTimeMinutes int       `json:"travel_time_minutes"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// RouteSegment represents a single hop within an alternative route.
type RouteSegment struct {
	FromTerminalID       uuid.UUID `json:"from_terminal_id"`
	ToTerminalID         uuid.UUID `json:"to_terminal_id"`
	TravelTimeMinutes    int       `json:"travel_time_minutes"`
	EstimatedWaitMinutes int       `json:"estimated_wait_minutes,omitempty"`
	QueueConfidence      float32   `json:"queue_confidence,omitempty"`
}

// AlternativeRoute represents one possible path from origin to destination.
type AlternativeRoute struct {
	RouteID                  string        `json:"route_id"`
	TotalTravelTimeMinutes   int           `json:"total_travel_time_minutes"`
	TotalEstimatedWaitMinutes int          `json:"total_estimated_wait_minutes"`
	TotalTimeMinutes         int           `json:"total_time_minutes"`
	Segments                 []RouteSegment `json:"segments"`
	HopCount                 int           `json:"hop_count"`
}

