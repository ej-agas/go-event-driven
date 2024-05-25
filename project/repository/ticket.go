package repository

import (
	"context"
	"fmt"

	"tickets/entities"
)

type TicketRepository struct {
	db Database
}

func NewTicketRepository(db Database) *TicketRepository {
	return &TicketRepository{db}
}

func (repository *TicketRepository) SaveTicketBooking(ctx context.Context, ticket entities.TicketBookingConfirmed) error {
	q := `INSERT INTO tickets (ticket_id, price_amount, price_currency, customer_email) VALUES ($1, $2, $3, $4);`

	_, err := repository.db.Exec(ctx, q, ticket.TicketID, ticket.Price.Amount, ticket.Price.Currency, ticket.CustomerEmail)
	if err != nil {
		return fmt.Errorf("error saving ticket: %w", err)
	}

	return nil
}
