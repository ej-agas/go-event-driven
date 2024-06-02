package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"tickets/entities"
	"tickets/message/event"
	"tickets/message/outbox"
)

type BookingRepository struct {
	db    *pgxpool.Pool
	stdDB *sql.DB
}

func NewBookingRepository(db *pgxpool.Pool, stdDB *sql.DB) *BookingRepository {
	if db == nil {
		panic("db passed to 'NewBookingRepository()' is nil!")
	}
	return &BookingRepository{db: db, stdDB: stdDB}
}

func (r *BookingRepository) Create(ctx context.Context, b entities.Booking) error {
	q := `
	INSERT INTO bookings (
	  id, 
	  show_id,
	  number_of_tickets,
	  customer_email
  ) VALUES ($1, $2, $3, $4)`

	tx, err := r.stdDB.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("error: could not begin transaction: %w", err)
	}

	defer func() {
		if err != nil {
			rollbackErr := tx.Rollback()
			err = errors.Join(err, rollbackErr)
			return
		}
		err = tx.Commit()
	}()

	if _, err := tx.ExecContext(ctx, q, b.ID, b.ShowID, b.NumberOfTickets, b.CustomerEmail); err != nil {
		return fmt.Errorf("error: failed to insert booking: %w", err)
	}

	outboxPublisher, err := outbox.NewPublisherForDb(ctx, tx)
	if err != nil {
		return fmt.Errorf("could not create event bus: %w", err)
	}

	err = event.NewEventBus(outboxPublisher).Publish(ctx, entities.BookingMade{
		Header:          entities.NewEventHeader(),
		BookingID:       b.ID,
		NumberOfTickets: b.NumberOfTickets,
		CustomerEmail:   b.CustomerEmail,
		ShowId:          b.ShowID,
	})

	if err != nil {
		return fmt.Errorf("could not publish event: %w", err)
	}

	return nil
}
