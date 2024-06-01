package http

import (
	"context"

	"tickets/entities"

	"github.com/ThreeDotsLabs/watermill/components/cqrs"
)

type Handler struct {
	eventBus              *cqrs.EventBus
	spreadsheetsAPIClient SpreadsheetsAPI
	ticketRepository      TicketRepository
	showRepository        ShowRepository
	bookingRepository     BookingRepository
}

type SpreadsheetsAPI interface {
	AppendRow(ctx context.Context, spreadsheetName string, row []string) error
}

type TicketRepository interface {
	All(ctx context.Context) ([]entities.Ticket, error)
}

type ShowRepository interface {
	Create(ctx context.Context, show entities.Show) error
}

type BookingRepository interface {
	Create(ctx context.Context, booking entities.Booking) error
}
