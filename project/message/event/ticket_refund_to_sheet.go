package event

import (
	"context"
	"fmt"

	"tickets/entities"

	"github.com/ThreeDotsLabs/go-event-driven/common/log"
)

type CancelTicketHandler struct {
	service SpreadsheetsAPI
}

func NewCancelTicketHandler(service SpreadsheetsAPI) *CancelTicketHandler {
	return &CancelTicketHandler{service: service}
}

func (handler *CancelTicketHandler) HandlerName() string {
	return "CancelTicket"
}

func (handler *CancelTicketHandler) NewEvent() interface{} {
	return &entities.TicketBookingCanceled{}
}

func (handler *CancelTicketHandler) Handle(ctx context.Context, event any) error {
	ticketBooking, ok := event.(*entities.TicketBookingCanceled)
	if !ok {
		return fmt.Errorf("unexpected event type: %T", event)
	}
	log.FromContext(ctx).Info("Issuing receipt: '%s'", ticketBooking.TicketID)

	return handler.service.AppendRow(
		ctx,
		"tickets-to-refund",
		[]string{
			ticketBooking.TicketID,
			ticketBooking.CustomerEmail,
			ticketBooking.Price.Amount,
			ticketBooking.Price.Currency,
		},
	)
}
