package event

import (
	"context"
	"fmt"

	"tickets/entities"

	"github.com/ThreeDotsLabs/go-event-driven/common/log"
)

type SaveToDatabaseHandler struct {
	repository TicketsRepository
}

func NewSaveToDatabaseHandler(repository TicketsRepository) *SaveToDatabaseHandler {
	return &SaveToDatabaseHandler{repository}
}

func (handler *SaveToDatabaseHandler) HandlerName() string {
	return "SaveToDatabase"
}

func (handler *SaveToDatabaseHandler) NewEvent() interface{} {
	return &entities.TicketBookingConfirmed{}
}

func (handler *SaveToDatabaseHandler) Handle(ctx context.Context, event any) error {
	log.FromContext(ctx).Info("Saving ticket to database")

	ticketBooking, ok := event.(*entities.TicketBookingConfirmed)
	if !ok {
		return fmt.Errorf("unexpected event type: %T", event)
	}

	return handler.repository.Save(ctx, &entities.Ticket{
		ID:            ticketBooking.TicketID,
		Price:         ticketBooking.Price,
		CustomerEmail: ticketBooking.CustomerEmail,
	})
}
