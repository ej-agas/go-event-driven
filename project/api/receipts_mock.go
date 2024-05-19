package api

import (
	"context"
	"sync"
	"time"

	"tickets/entities"
)

type ReceiptsServiceMock struct {
	lock           sync.Mutex
	IssuedReceipts []entities.IssueReceiptRequest
}

func (r *ReceiptsServiceMock) IssueReceipt(ctx context.Context, request entities.IssueReceiptRequest) (entities.IssueReceiptResponse, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.IssuedReceipts = append(r.IssuedReceipts, request)

	return entities.IssueReceiptResponse{
		ReceiptNumber: "mocked-receipt-number",
		IssuedAt:      time.Now(),
	}, nil
}
