package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/ThreeDotsLabs/go-event-driven/common/clients"
	"github.com/ThreeDotsLabs/go-event-driven/common/clients/receipts"
	"github.com/ThreeDotsLabs/go-event-driven/common/clients/spreadsheets"
	commonHTTP "github.com/ThreeDotsLabs/go-event-driven/common/http"
	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/labstack/echo/v4"
	"github.com/lithammer/shortuuid/v3"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

var malformedEventID = "2beaf5bc-d5e4-4653-b075-2b36bbf28949"

type Ticket struct {
	ID            string `json:"ticket_id"`
	Status        string `json:"status"`
	Price         Price  `json:"price"`
	CustomerEmail string `json:"customer_email"`
}

type Price struct {
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
}

type TicketsStatusRequest struct {
	Tickets []Ticket `json:"tickets"`
}

type EventHeader struct {
	ID          string    `json:"id"`
	PublishedAt time.Time `json:"published_at"`
}

type TicketBookingConfirmed struct {
	Header        EventHeader `json:"header"`
	TicketID      string      `json:"ticket_id"`
	CustomerEmail string      `json:"customer_email"`
	Price         Price       `json:"price"`
}

type TicketBookingCanceled struct {
	Header        EventHeader `json:"header"`
	TicketID      string      `json:"ticket_id"`
	CustomerEmail string      `json:"customer_email"`
	Price         Price       `json:"price"`
}

func NewEventHeader() EventHeader {
	return EventHeader{
		ID:          watermill.NewUUID(),
		PublishedAt: time.Now().UTC(),
	}
}

func main() {
	log.Init(logrus.InfoLevel)

	clients, err := clients.NewClients(
		os.Getenv("GATEWAY_ADDR"),
		func(ctx context.Context, req *http.Request) error {
			req.Header.Set("Correlation-ID", log.CorrelationIDFromContext(ctx))
			return nil
		},
	)
	if err != nil {
		panic(err)
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_ADDR"),
	})

	logger := log.NewWatermill(logrus.NewEntry(logrus.StandardLogger()))

	publisher, err := redisstream.NewPublisher(redisstream.PublisherConfig{
		Client: redisClient,
	}, logger)

	sub, err := redisstream.NewSubscriber(redisstream.SubscriberConfig{
		Client: redisClient,
	}, logger)

	router, err := message.NewRouter(message.RouterConfig{}, logger)

	if err != nil {
		panic(err)
	}

	retryMiddleware := middleware.Retry{
		MaxRetries:      10,
		InitialInterval: time.Millisecond * 100,
		MaxInterval:     time.Second,
		Multiplier:      2,
		Logger:          logger,
	}

	router.AddMiddleware(CorrelationIDMiddleware)
	router.AddMiddleware(LoggingMiddleware)
	router.AddMiddleware(retryMiddleware.Middleware)

	worker := NewWorker(publisher, sub, NewReceiptsClient(clients), NewSpreadsheetsClient(clients), router)
	go worker.ProcessIssueReceiptMessages()
	go worker.ProcessAppendToTrackerMessages()
	go worker.ProcessTicketsToRefund()

	e := commonHTTP.NewEcho()
	e.POST("/tickets-status", func(c echo.Context) error {
		var request TicketsStatusRequest
		err := c.Bind(&request)
		if err != nil {
			return err
		}

		correlationID := c.Request().Header.Get("Correlation-ID")

		for _, ticket := range request.Tickets {
			var topic string

			event := createEvent(ticket)

			switch event.(type) {
			case TicketBookingConfirmed:
				topic = "TicketBookingConfirmed"
			case TicketBookingCanceled:
				topic = "TicketBookingCanceled"
			default:
				return errors.New("invalid ticket booking event")
			}

			if err := publishMessage(worker, topic, event, correlationID); err != nil {
				return err
			}
		}

		return c.NoContent(http.StatusOK)
	})

	e.GET("/health", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	logrus.Info("Server starting...")

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	errgrp, ctx := errgroup.WithContext(ctx)

	// start router in separate goroutine
	errgrp.Go(func() error {
		return router.Run(ctx)
	})

	// start http server in separate goroutine
	errgrp.Go(func() error {
		// wait for watermill router to run before starting http server
		<-router.Running()

		err := e.Start(":8080")
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}

		return nil
	})

	// wait for os interrupt before shutting down server gracefully
	errgrp.Go(func() error {
		<-ctx.Done()
		return e.Shutdown(ctx)
	})

	err = errgrp.Wait()
	if err != nil {
		panic(err)
	}
}

func createEvent(ticket Ticket) interface{} {
	header := NewEventHeader()

	if ticket.Status == "canceled" {
		return TicketBookingCanceled{
			Header:        header,
			TicketID:      ticket.ID,
			CustomerEmail: ticket.CustomerEmail,
			Price:         ticket.Price,
		}
	}

	return TicketBookingConfirmed{
		Header:        header,
		TicketID:      ticket.ID,
		CustomerEmail: ticket.CustomerEmail,
		Price:         ticket.Price,
	}
}

func publishMessage(worker *Worker, topic string, event interface{}, correlationID string) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	msg := message.NewMessage(watermill.NewUUID(), payload)
	msg.Metadata.Set("correlation_id", correlationID)
	msg.Metadata.Set("type", topic)

	worker.Send(topic, msg)

	return nil
}

type ReceiptsClient struct {
	clients *clients.Clients
}

func NewReceiptsClient(clients *clients.Clients) ReceiptsClient {
	return ReceiptsClient{
		clients: clients,
	}
}

type IssueReceiptRequest struct {
	TicketID string
	Price    receipts.Money
}

func (c ReceiptsClient) IssueReceipt(ctx context.Context, request IssueReceiptRequest) error {
	body := receipts.PutReceiptsJSONRequestBody{
		TicketId: request.TicketID,
		Price:    request.Price,
	}

	receiptsResp, err := c.clients.Receipts.PutReceiptsWithResponse(ctx, body)
	if err != nil {
		return err
	}
	if receiptsResp.StatusCode() != http.StatusOK {
		return fmt.Errorf("unexpected status code: %v", receiptsResp.StatusCode())
	}

	return nil
}

type SpreadsheetsClient struct {
	clients *clients.Clients
}

func NewSpreadsheetsClient(clients *clients.Clients) SpreadsheetsClient {
	return SpreadsheetsClient{
		clients: clients,
	}
}

func (c SpreadsheetsClient) AppendRow(ctx context.Context, spreadsheetName string, row []string) error {
	request := spreadsheets.PostSheetsSheetRowsJSONRequestBody{
		Columns: row,
	}

	sheetsResp, err := c.clients.Spreadsheets.PostSheetsSheetRowsWithResponse(ctx, spreadsheetName, request)
	if err != nil {
		return err
	}
	if sheetsResp.StatusCode() != http.StatusOK {
		return fmt.Errorf("unexpected status code: %v", sheetsResp.StatusCode())
	}

	return nil
}

type Worker struct {
	publisher          *redisstream.Publisher
	subscriber         *redisstream.Subscriber
	receiptsClient     ReceiptsClient
	spreadSheetsClient SpreadsheetsClient
	router             *message.Router
}

func NewWorker(
	publisher *redisstream.Publisher,
	subscriber *redisstream.Subscriber,
	receiptsClient ReceiptsClient,
	spreadsheetsClient SpreadsheetsClient,
	router *message.Router,
) *Worker {
	return &Worker{
		publisher:          publisher,
		subscriber:         subscriber,
		receiptsClient:     receiptsClient,
		spreadSheetsClient: spreadsheetsClient,
		router:             router,
	}
}

func (w *Worker) ProcessIssueReceiptMessages() {
	w.router.AddNoPublisherHandler(
		"receipt-messages-handler",
		"TicketBookingConfirmed",
		w.subscriber,
		func(msg *message.Message) error {
			var payload TicketBookingConfirmed

			if msg.UUID == malformedEventID {
				return nil
			}

			if err := json.Unmarshal(msg.Payload, &payload); err != nil {
				return err
			}

			if payload.Price.Currency == "" {
				payload.Price.Currency = "USD"
			}

			issueReceiptRequest := IssueReceiptRequest{
				TicketID: payload.TicketID,
				Price: receipts.Money{
					MoneyAmount:   payload.Price.Amount,
					MoneyCurrency: payload.Price.Currency,
				},
			}

			if err := w.receiptsClient.IssueReceipt(msg.Context(), issueReceiptRequest); err != nil {
				logrus.WithError(err).Error("failed to issue the receipt")
				return err
			}
			return nil
		},
	)
}

func (w *Worker) ProcessAppendToTrackerMessages() {
	w.router.AddNoPublisherHandler(
		"append-to-tracker-handler",
		"TicketBookingConfirmed",
		w.subscriber,
		func(msg *message.Message) error {
			var payload TicketBookingConfirmed

			if msg.UUID == "2beaf5bc-d5e4-4653-b075-2b36bbf28949" {
				return nil
			}

			if msg.Metadata.Get("type") != "TicketBookingConfirmed" {
				return nil
			}

			if err := json.Unmarshal(msg.Payload, &payload); err != nil {
				return err
			}

			if payload.Price.Currency == "" {
				payload.Price.Currency = "USD"
			}

			if err := w.spreadSheetsClient.AppendRow(
				msg.Context(),
				"tickets-to-print",
				[]string{payload.TicketID, payload.CustomerEmail, payload.Price.Amount, payload.Price.Currency},
			); err != nil {
				logrus.WithError(err).Error("failed to append to tracker")
				return err
			}
			return nil
		},
	)
}

func (w *Worker) ProcessTicketsToRefund() {
	w.router.AddNoPublisherHandler(
		"process-tickets-to-refund-handler",
		"TicketBookingCanceled",
		w.subscriber,
		func(msg *message.Message) error {
			var payload TicketBookingCanceled

			if msg.UUID == malformedEventID {
				return nil
			}

			if msg.Metadata.Get("type") != "TicketBookingCanceled" {
				return nil
			}

			if err := json.Unmarshal(msg.Payload, &payload); err != nil {
				return err
			}

			if payload.Price.Currency == "" {
				payload.Price.Currency = "USD"
			}

			if err := w.spreadSheetsClient.AppendRow(
				msg.Context(),
				"tickets-to-refund",
				[]string{payload.TicketID, payload.CustomerEmail, payload.Price.Amount, payload.Price.Currency},
			); err != nil {
				logrus.WithError(err).Error("failed to append to tracker")
				return err
			}
			return nil
		},
	)
}

func (w *Worker) Send(topic string, msg ...*message.Message) {
	w.publisher.Publish(topic, msg...)
}

func CorrelationIDMiddleware(next message.HandlerFunc) message.HandlerFunc {
	return func(msg *message.Message) ([]*message.Message, error) {
		ctx := msg.Context()

		correlationID := msg.Metadata.Get("correlation_id")
		if correlationID == "" {
			correlationID = shortuuid.New()
		}

		ctx = log.ToContext(ctx, logrus.WithFields(logrus.Fields{"correlation_id": correlationID}))
		ctx = log.ContextWithCorrelationID(ctx, correlationID)

		msg.SetContext(ctx)

		return next(msg)
	}
}

func LoggingMiddleware(next message.HandlerFunc) message.HandlerFunc {
	return func(msg *message.Message) ([]*message.Message, error) {
		logger := log.FromContext(msg.Context())
		logger = logger.WithField("message_uuid", msg.UUID)

		msgs, err := next(msg)

		if err != nil {
			logger.WithError(err).Error("Message handling error")
		}

		logger.Info("Handling a message")

		return msgs, err
	}
}
