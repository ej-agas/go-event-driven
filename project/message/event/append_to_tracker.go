package event

import (
	"context"
	"fmt"

	"tickets/entities"

	"github.com/ThreeDotsLabs/go-event-driven/common/log"
)

type AppendToTrackerHandler struct {
	service SpreadsheetsAPI
}

func NewAppendToTrackerHandler(service SpreadsheetsAPI) *AppendToTrackerHandler {
	return &AppendToTrackerHandler{service: service}
}

func (handler *AppendToTrackerHandler) HandlerName() string {
	return "AppendToTracker"
}

func (handler *AppendToTrackerHandler) NewEvent() interface{} {
	return &entities.TicketBookingConfirmed{}
}

func (handler *AppendToTrackerHandler) Handle(ctx context.Context, event any) error {
	log.FromContext(ctx).Info("Appending ticket to the tracker")

	ticketBooking, ok := event.(*entities.TicketBookingConfirmed)
	if !ok {
		return fmt.Errorf("unexpected event type: %T", event)
	}

	return handler.service.AppendRow(
		ctx,
		"tickets-to-print",
		[]string{
			ticketBooking.TicketID,
			ticketBooking.CustomerEmail,
			ticketBooking.Price.Amount,
			ticketBooking.Price.Currency,
		},
	)
}
