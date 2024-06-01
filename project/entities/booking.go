package entities

import "github.com/google/uuid"

type Booking struct {
	ID              uuid.UUID `json:"id"`
	ShowID          uuid.UUID `json:"show_id"`
	NumberOfTickets int       `json:"number_of_tickets"`
	CustomerEmail   string    `json:"customer_email"`
}
