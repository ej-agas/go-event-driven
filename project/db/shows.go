package db

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"tickets/entities"
)

type ShowRepository struct {
	db *pgxpool.Pool
}

func NewShowRepository(db *pgxpool.Pool) *ShowRepository {
	if db == nil {
		panic("db passed to 'NewShowRepository()' is nil!")
	}
	return &ShowRepository{db: db}
}

func (repository *ShowRepository) Create(ctx context.Context, show entities.Show) error {
	q := `INSERT INTO shows (
		 id,
         dead_nation_id,
		 number_of_tickets, 
		 start_time, 
		 title,
         venue
		 ) VALUES ($1, $2, $3, $4, $5, $6) 
		   ON CONFLICT DO NOTHING;
	`

	_, err := repository.db.Exec(
		ctx,
		q,
		show.ID,
		show.DeadNationID,
		show.NumberOfTickets,
		show.StartTime,
		show.Title,
		show.Venue,
	)

	if err != nil {
		return fmt.Errorf("error creating show: %w", err)
	}

	return nil
}
