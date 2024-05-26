package event

import (
	"context"
	"fmt"

	"tickets/entities"

	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
)

type SaveToFileHandler struct {
	api      FilesAPI
	eventBus *cqrs.EventBus
}

func NewSaveToFileHandler(api FilesAPI, eventBus *cqrs.EventBus) *SaveToFileHandler {
	return &SaveToFileHandler{api, eventBus}
}

func (handler *SaveToFileHandler) HandlerName() string {
	return "SaveToFile"
}

func (handler *SaveToFileHandler) NewEvent() interface{} {
	return &entities.TicketBookingConfirmed{}
}

func (handler *SaveToFileHandler) Handle(ctx context.Context, event any) error {
	log.FromContext(ctx).Info("Saving ticket to file")

	ticketBooking, ok := event.(*entities.TicketBookingConfirmed)
	if !ok {
		return fmt.Errorf("unexpected event type: %T", event)
	}

	body := `
		<!DOCTYPE html>
	<html lang="en">
	<head>
		<meta charset="UTF-8">
		<meta name="viewport" content="width=device-width, initial-scale=1.0">
		<title>Ticket -` + ticketBooking.TicketID + `</title>
		<meta name="description" content="receipt">
	</head>
	<body>
		<h1>Ticket - ` + ticketBooking.TicketID + `</h1>
		<h1>Price:` + ticketBooking.Price.Amount + ` ` + ticketBooking.Price.Currency + `</h1>
	</body>
	</html>
`

	fileName := fmt.Sprintf("%s-ticket.html", ticketBooking.TicketID)
	err := handler.api.Upload(ctx, fileName, body)
	if err != nil {
		return fmt.Errorf("save ticket booking failed: %w", err)
	}

	err = handler.eventBus.Publish(ctx, entities.TicketPrinted{
		Header:   entities.NewEventHeader(),
		TicketID: ticketBooking.TicketID,
		FileName: fileName,
	})

	if err != nil {
		return fmt.Errorf("failed to publish ticket printed event: %w", err)
	}

	return nil
}
