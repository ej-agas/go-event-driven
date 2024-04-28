package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

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
)

type TicketsConfirmationRequest struct {
	Tickets []string `json:"tickets"`
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

	worker := NewWorker(publisher, sub, NewReceiptsClient(clients), NewSpreadsheetsClient(clients))
	go worker.ProcessIssueReceiptMessages()
	go worker.ProcessAppendToTrackerMessages()

	e := commonHTTP.NewEcho()

	e.POST("/tickets-confirmation", func(c echo.Context) error {
		var request TicketsConfirmationRequest
		err := c.Bind(&request)
		if err != nil {
			return err
		}

		for _, ticket := range request.Tickets {
			worker.Send("issue-receipt", message.NewMessage(watermill.NewUUID(), []byte(ticket)))
			worker.Send("append-to-tracker", message.NewMessage(watermill.NewUUID(), []byte(ticket)))
		}

		return c.NoContent(http.StatusOK)
	})

	logrus.Info("Server starting...")

	err = e.Start(":8080")
	if err != nil && err != http.ErrServerClosed {
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

func (c ReceiptsClient) IssueReceipt(ctx context.Context, ticketID string) error {
	body := receipts.PutReceiptsJSONRequestBody{
		TicketId: ticketID,
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
}

func NewWorker(
	publisher *redisstream.Publisher,
	subscriber *redisstream.Subscriber,
	receiptsClient ReceiptsClient,
	spreadsheetsClient SpreadsheetsClient,
) *Worker {
	return &Worker{
		publisher:          publisher,
		subscriber:         subscriber,
		receiptsClient:     receiptsClient,
		spreadSheetsClient: spreadsheetsClient,
	}
}

func (w *Worker) ProcessIssueReceiptMessages() {
	messages, err := w.subscriber.Subscribe(context.Background(), "issue-receipt")
	if err != nil {
		panic(err)
	}

	for msg := range messages {
		if err := w.receiptsClient.IssueReceipt(msg.Context(), string(msg.Payload)); err != nil {
			logrus.WithError(err).Error("failed to issue the receipt")
			msg.Nack()
			continue
		}

		msg.Ack()
	}
}

func (w *Worker) ProcessAppendToTrackerMessages() {
	messages, err := w.subscriber.Subscribe(context.Background(), "append-to-tracker")
	if err != nil {
		panic(err)
	}

	for msg := range messages {
		if err := w.spreadSheetsClient.AppendRow(msg.Context(), "tickets-to-print", []string{string(msg.Payload)}); err != nil {
			logrus.WithError(err).Error("failed to append to tracker")
			msg.Nack()
			continue
		}

		msg.Ack()
	}
}

func (w *Worker) Send(topic string, msg ...*message.Message) {
	w.publisher.Publish(topic, msg...)
}
