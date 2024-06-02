package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"

	"tickets/api"
	"tickets/message"
	"tickets/service"

	"database/sql"
	"github.com/ThreeDotsLabs/go-event-driven/common/clients"
	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/lib/pq"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	apiClients, err := clients.NewClients(
		os.Getenv("GATEWAY_ADDR"),
		func(ctx context.Context, req *http.Request) error {
			req.Header.Set("Correlation-ID", log.CorrelationIDFromContext(ctx))
			return nil
		},
	)
	if err != nil {
		panic(err)
	}

	redisClient := message.NewRedisClient(os.Getenv("REDIS_ADDR"))
	defer redisClient.Close()

	spreadsheetsService := api.NewSpreadsheetsAPIClient(apiClients)
	receiptsService := api.NewReceiptsServiceClient(apiClients)
	filesAPI := api.NewFilesAPIClient(apiClients)

	postgres, err := pgxpool.New(context.Background(), os.Getenv("POSTGRES_URL"))
	if err != nil {
		panic(err)
	}
	defer postgres.Close()

	stdLibDB, err := sql.Open("postgres", os.Getenv("POSTGRES_URL"))

	err = service.New(
		redisClient,
		postgres,
		stdLibDB,
		spreadsheetsService,
		receiptsService,
		filesAPI,
	).Run(ctx)
	if err != nil {
		panic(err)
	}
}
