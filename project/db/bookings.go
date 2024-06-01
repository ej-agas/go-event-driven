package db

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"tickets/entities"
)

type BookingRepository struct {
	db *pgxpool.Pool
}

func NewBookingRepository(db *pgxpool.Pool) *BookingRepository {
	if db == nil {
		panic("db passed to 'NewBookingRepository()' is nil!")
	}
	return &BookingRepository{db: db}
}

func (r BookingRepository) Create(ctx context.Context, b entities.Booking) error {
	q := `
	INSERT INTO bookings (
	  id, 
	  show_id,
	  number_of_tickets,
	  customer_email
  ) VALUES ($1, $2, $3, $4)`

	if _, err := r.db.Exec(ctx, q, b.ID, b.ShowID, b.NumberOfTickets, b.CustomerEmail); err != nil {
		return fmt.Errorf("error: failed to insert booking: %w", err)
	}

	return nil
}
