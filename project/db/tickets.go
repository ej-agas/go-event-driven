package db

import (
	"context"
	"fmt"

	"tickets/entities"

	"github.com/jackc/pgx/v5/pgxpool"
)

type TicketRepository struct {
	db *pgxpool.Pool
}

func NewTicketRepository(db *pgxpool.Pool) *TicketRepository {
	return &TicketRepository{db}
}

func (repository *TicketRepository) Save(ctx context.Context, ticket *entities.Ticket) error {
	q := `INSERT INTO tickets (ticket_id, price_amount, price_currency, customer_email) VALUES ($1, $2, $3, $4);`

	_, err := repository.db.Exec(ctx, q, ticket.ID, ticket.Price.Amount, ticket.Price.Currency, ticket.CustomerEmail)
	if err != nil {
		return fmt.Errorf("error saving ticket: %w", err)
	}

	return nil
}

func (repository *TicketRepository) Delete(ctx context.Context, id string) error {
	_, err := repository.db.Exec(ctx, "DELETE FROM tickets WHERE ticket_id = $1;", id)

	if err != nil {
		return fmt.Errorf("error deleting ticket: %w", err)
	}

	return nil
}
