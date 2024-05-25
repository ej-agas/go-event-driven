package event

import (
	"context"
	"fmt"

	"tickets/entities"

	"github.com/ThreeDotsLabs/go-event-driven/common/log"
)

type DeleteCanceledTicketsHandler struct {
	repository TicketsRepository
}

func NewDeleteCanceledTicketsHandler(repository TicketsRepository) *DeleteCanceledTicketsHandler {
	return &DeleteCanceledTicketsHandler{repository: repository}
}

func (handler *DeleteCanceledTicketsHandler) HandlerName() string {
	return "DeleteCanceledTickets"
}

func (handler *DeleteCanceledTicketsHandler) NewEvent() interface{} {
	return &entities.TicketBookingCanceled{}
}

func (handler *DeleteCanceledTicketsHandler) Handle(ctx context.Context, event any) error {
	log.FromContext(ctx).Info("Deleting tickets from database")

	ticketBooking, ok := event.(*entities.TicketBookingCanceled)
	if !ok {
		return fmt.Errorf("unexpected event type: %T", event)
	}

	return handler.repository.Delete(ctx, ticketBooking.TicketID)
}
