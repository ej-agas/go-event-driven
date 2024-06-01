package entities

import (
	"github.com/google/uuid"
	"time"
)

type Show struct {
	ID              uuid.UUID `json:"id"`
	DeadNationID    uuid.UUID `json:"dead_nation_id"`
	NumberOfTickets int       `json:"number_of_tickets"`
	StartTime       time.Time `json:"start_time"`
	Title           string    `json:"title"`
	Venue           string    `json:"venue"`
}
