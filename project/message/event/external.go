package event

import (
	"context"

	"tickets/entities"
)

type SpreadsheetsAPI interface {
	AppendRow(ctx context.Context, sheetName string, row []string) error
}

type ReceiptsService interface {
	IssueReceipt(ctx context.Context, request entities.IssueReceiptRequest) (entities.IssueReceiptResponse, error)
}

type TicketsRepository interface {
	Save(ctx context.Context, ticket *entities.Ticket) error
	Delete(ctx context.Context, ticketID string) error
}

type FilesAPI interface {
	Upload(ctx context.Context, name, contents string) error
	Download(ctx context.Context, name string) (string, error)
}
