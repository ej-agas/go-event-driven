package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func CreateDatabaseSchema(db *pgxpool.Pool) error {
	_, err := db.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS tickets (
			ticket_id UUID PRIMARY KEY,
			price_amount NUMERIC(10, 2) NOT NULL,
			price_currency CHAR(3) NOT NULL,
			customer_email VARCHAR(255) NOT NULL
		);

		CREATE TABLE IF NOT EXISTS shows (
		    id UUID PRIMARY KEY,
		    dead_nation_id VARCHAR(255) NOT NULL,
		    number_of_tickets INTEGER NOT NULL,
		    start_time TIMESTAMP NOT NULL,
		    title VARCHAR(255) NOT NULL,
		    venue VARCHAR(255) NOT NULL
		);

		CREATE TABLE IF NOT EXISTS bookings (
		    id UUID PRIMARY KEY,
		    show_id UUID NOT NULL,
		    number_of_tickets INTEGER NOT NULL,
		    customer_email VARCHAR(255) NOT NULL
		)
	`)
	if err != nil {
		return fmt.Errorf("pgxpool error: error executing create table query: %w", err)
	}

	return nil
}
