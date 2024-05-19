package tests_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"

	"tickets/api"
	"tickets/entities"
	"tickets/message"
	"tickets/service"

	"github.com/google/uuid"
	"github.com/lithammer/shortuuid/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComponent(t *testing.T) {
	redisClient := message.NewRedisClient(os.Getenv("REDIS_ADDR"))
	defer redisClient.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	spreadsheetsService := &api.SpreadsheetsAPIMock{}
	receiptsService := &api.ReceiptsServiceMock{}

	go func() {
		svc := service.New(
			redisClient,
			spreadsheetsService,
			receiptsService,
		)
		assert.NoError(t, svc.Run(ctx))
	}()

	waitForHttpServer(t)

	ticket := TicketStatus{
		TicketID: uuid.NewString(),
		Status:   "confirmed",
		Price: Money{
			Amount:   "50.30",
			Currency: "GBP",
		},
		Email:     "email@example.com",
		BookingID: uuid.NewString(),
	}

	sendTicketsStatus(t, TicketsStatusRequest{Tickets: []TicketStatus{ticket}})

	assertReceiptForTicketIssued(t, receiptsService, ticket)
	assertRowToSheetAdded(t, spreadsheetsService, ticket, "tickets-to-print")

	failedTicket := TicketStatus{
		TicketID: uuid.NewString(),
		Status:   "canceled",
		Price: Money{
			Amount:   "50.30",
			Currency: "GBP",
		},
		Email:     "email@example.com",
		BookingID: uuid.NewString(),
	}

	sendTicketsStatus(t, TicketsStatusRequest{Tickets: []TicketStatus{failedTicket}})
	assertRowToSheetAdded(t, spreadsheetsService, failedTicket, "tickets-to-refund")
}

func waitForHttpServer(t *testing.T) {
	t.Helper()

	require.EventuallyWithT(
		t,
		func(t *assert.CollectT) {
			resp, err := http.Get("http://localhost:8080/health")
			if !assert.NoError(t, err) {
				return
			}
			defer resp.Body.Close()

			if assert.Less(t, resp.StatusCode, 300, "API not ready, http status: %d", resp.StatusCode) {
				return
			}
		},
		time.Second*10,
		time.Millisecond*50,
	)
}

type TicketsStatusRequest struct {
	Tickets []TicketStatus `json:"tickets"`
}

type TicketStatus struct {
	TicketID  string `json:"ticket_id"`
	Status    string `json:"status"`
	Price     Money  `json:"price"`
	Email     string `json:"email"`
	BookingID string `json:"booking_id"`
}

type Money struct {
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
}

func sendTicketsStatus(t *testing.T, req TicketsStatusRequest) {
	t.Helper()

	payload, err := json.Marshal(req)
	require.NoError(t, err)

	correlationID := shortuuid.New()

	ticketIDs := make([]string, 0, len(req.Tickets))
	for _, ticket := range req.Tickets {
		ticketIDs = append(ticketIDs, ticket.TicketID)
	}

	httpReq, err := http.NewRequest(
		http.MethodPost,
		"http://localhost:8080/tickets-status",
		bytes.NewBuffer(payload),
	)
	require.NoError(t, err)

	httpReq.Header.Set("Correlation-ID", correlationID)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func assertReceiptForTicketIssued(t *testing.T, receiptsService *api.ReceiptsServiceMock, ticket TicketStatus) {
	assert.EventuallyWithT(
		t,
		func(collectT *assert.CollectT) {
			issuedReceipts := len(receiptsService.IssuedReceipts)
			t.Log("issued receipts", issuedReceipts)

			assert.Greater(collectT, issuedReceipts, 0, "no receipts issued")
		},
		10*time.Second,
		100*time.Millisecond,
	)

	var receipt entities.IssueReceiptRequest
	var ok bool
	for _, issuedReceipt := range receiptsService.IssuedReceipts {
		if issuedReceipt.TicketID != ticket.TicketID {
			continue
		}
		receipt = issuedReceipt
		ok = true
		break
	}
	require.Truef(t, ok, "receipt for ticket %s not found", ticket.TicketID)

	assert.Equal(t, ticket.TicketID, receipt.TicketID)
	assert.Equal(t, ticket.Price.Amount, receipt.Price.Amount)
	assert.Equal(t, ticket.Price.Currency, receipt.Price.Currency)
}

func assertRowToSheetAdded(t *testing.T, spreadsheetsService *api.SpreadsheetsAPIMock, ticket TicketStatus, sheetName string) bool {
	return assert.EventuallyWithT(
		t,
		func(t *assert.CollectT) {
			rows, ok := spreadsheetsService.Rows[sheetName]
			if !assert.True(t, ok, "sheet %s not found", sheetName) {
				return
			}

			allValues := []string{}

			for _, row := range rows {
				for _, col := range row {
					allValues = append(allValues, col)
				}
			}

			assert.Contains(t, allValues, ticket.TicketID, "ticket id not found in sheet %s", sheetName)
		},
		10*time.Second,
		100*time.Millisecond,
	)
}
