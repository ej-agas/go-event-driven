package entities

import (
	"time"
)

type IssueReceiptRequest struct {
	IdempotencyKey string
	TicketID       string
	Price          Price
}

type IssueReceiptResponse struct {
	ReceiptNumber string    `json:"number"`
	IssuedAt      time.Time `json:"issued_at"`
}
