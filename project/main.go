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
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

type TicketStatus struct {
	TicketID      string `json:"ticket_id"`
	Status        string `json:"status"`
	Price         Price  `json:"price"`
	CustomerEmail string `json:"customer_email"`
}

type Price struct {
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
}

type TicketsStatusRequest struct {
	Tickets []TicketStatus `json:"tickets"`
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

	clients, err := clients.NewClients(os.Getenv("GATEWAY_ADDR"), nil)
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

		for _, ticket := range request.Tickets {
			var event interface{}
			topic := "TicketBookingConfirmed"

			if ticket.Status == "canceled" {
				event = TicketBookingCanceled{
					Header:        NewEventHeader(),
					TicketID:      ticket.TicketID,
					CustomerEmail: ticket.CustomerEmail,
					Price:         ticket.Price,
				}
				topic = "TicketBookingCanceled"
			} else {
				event = TicketBookingConfirmed{
					Header:        NewEventHeader(),
					TicketID:      ticket.TicketID,
					CustomerEmail: ticket.CustomerEmail,
					Price:         ticket.Price,
				}
			}

			payload, err := json.Marshal(event)
			if err != nil {
				return err
			}

			worker.Send(topic, message.NewMessage(watermill.NewUUID(), payload))
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

			if err := json.Unmarshal(msg.Payload, &payload); err != nil {
				return err
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

			if err := json.Unmarshal(msg.Payload, &payload); err != nil {
				return err
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

			if err := json.Unmarshal(msg.Payload, &payload); err != nil {
				return err
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
