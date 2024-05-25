package db

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"

	"tickets/entities"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
)

var db *pgxpool.Pool
var getDbOnce sync.Once

func getDb() *pgxpool.Pool {
	getDbOnce.Do(func() {
		var err error
		db, err = pgxpool.New(context.Background(), os.Getenv("POSTGRES_URL"))
		if err != nil {
			panic(err)
		}

		if err := CreateDatabaseSchema(db); err != nil {
			panic(err)
		}
	})
	return db
}

func TestForIdempotency(t *testing.T) {
	repository := NewTicketRepository(getDb())
	fmt.Printf("%#v\n", repository)
	ticket := entities.Ticket{
		ID:     "8e7de3b9-9209-42eb-ab3d-886402ee45d2",
		Status: "completed",
		Price: entities.Price{
			Amount:   "100.25",
			Currency: "PHP",
		},
		CustomerEmail: "customer@example.com",
	}

	err := repository.Save(context.Background(), &ticket)
	assert.NoError(t, err)
	err2 := repository.Save(context.Background(), &ticket)
	assert.NoError(t, err2)

	all, err3 := repository.All(context.Background())
	fmt.Println(all)
	assert.NoError(t, err3)

	assert.Equal(t, 1, len(all))
	assert.Equal(t, ticket.ID, all[0].ID)
}
