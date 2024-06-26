package service

import (
	"context"
	"database/sql"
	"fmt"
	stdHTTP "net/http"
	"tickets/message/outbox"

	"tickets/db"
	ticketsHttp "tickets/http"
	"tickets/message"
	"tickets/message/event"

	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	watermillMessage "github.com/ThreeDotsLabs/watermill/message"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

func init() {
	log.Init(logrus.InfoLevel)
}

type Service struct {
	db              *pgxpool.Pool
	watermillRouter *watermillMessage.Router
	echoRouter      *echo.Echo
}

func New(
	redisClient *redis.Client,
	postgres *pgxpool.Pool,
	stdDB *sql.DB,
	spreadsheetsService event.SpreadsheetsAPI,
	receiptsService event.ReceiptsService,
	filesService event.FilesAPI,
) Service {
	watermillLogger := log.NewWatermill(log.FromContext(context.Background()))

	var redisPublisher watermillMessage.Publisher
	redisPublisher = message.NewRedisPublisher(redisClient, watermillLogger)
	redisPublisher = log.CorrelationPublisherDecorator{Publisher: redisPublisher}

	eventBus := event.NewEventBus(redisPublisher)

	watermillRouter := message.NewWatermillRouter(
		watermillLogger,
	)

	ticketRepository := db.NewTicketRepository(postgres)
	showRepository := db.NewShowRepository(postgres)
	bookingRepository := db.NewBookingRepository(postgres, stdDB)

	postgresSubscriber := outbox.NewPostgresSubscriber(stdDB, watermillLogger)
	outbox.AddForwarderHandler(postgresSubscriber, redisPublisher, watermillRouter, watermillLogger)

	eventProcessorConfig := event.NewProcessorConfig(redisClient, watermillLogger)
	event.RegisterEventHandlers(
		watermillRouter,
		eventProcessorConfig,
		spreadsheetsService,
		receiptsService,
		ticketRepository,
		filesService,
		eventBus,
	)

	echoRouter := ticketsHttp.NewHttpRouter(
		eventBus,
		spreadsheetsService,
		ticketRepository,
		showRepository,
		bookingRepository,
	)

	return Service{
		postgres,
		watermillRouter,
		echoRouter,
	}
}

func (s Service) Run(
	ctx context.Context,
) error {
	if err := db.CreateDatabaseSchema(s.db); err != nil {
		return fmt.Errorf("error creating database schema: %w", err)
	}

	errgrp, ctx := errgroup.WithContext(ctx)

	errgrp.Go(func() error {
		return s.watermillRouter.Run(ctx)
	})

	errgrp.Go(func() error {
		// we don't want to start HTTP server before Watermill router (so service won't be healthy before it's ready)
		<-s.watermillRouter.Running()

		err := s.echoRouter.Start(":8080")

		if err != nil && err != stdHTTP.ErrServerClosed {
			return err
		}

		return nil
	})

	errgrp.Go(func() error {
		<-ctx.Done()
		return s.echoRouter.Shutdown(context.Background())
	})

	return errgrp.Wait()
}
