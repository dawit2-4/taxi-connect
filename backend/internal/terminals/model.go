package terminals

import (
	"time"
	"github.com/google/uuid"
)

type Terminal struct {
	ID   uuid.UUID  `json:"id"`
	Name string      `json:"name"`
	Location string  `json:"location"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

