package event

import (
	"context"
	"fmt"

	"tickets/entities"

	"github.com/ThreeDotsLabs/go-event-driven/common/log"
)

type IssueReceiptHandler struct {
	service ReceiptsService
}

func NewIssueReceiptHandler(service ReceiptsService) *IssueReceiptHandler {
	return &IssueReceiptHandler{service: service}
}

func (handler *IssueReceiptHandler) HandlerName() string {
	return "IssueReceipt"
}

func (handler *IssueReceiptHandler) NewEvent() interface{} {
	return &entities.TicketBookingConfirmed{}
}

func (handler *IssueReceiptHandler) Handle(ctx context.Context, event any) error {
	ticketBooking, ok := event.(*entities.TicketBookingConfirmed)
	if !ok {
		return fmt.Errorf("unexpected event type: %T", event)
	}
	log.FromContext(ctx).Info("Issuing receipt: '%s'", ticketBooking.TicketID)

	request := entities.IssueReceiptRequest{
		TicketID: ticketBooking.TicketID,
		Price:    ticketBooking.Price,
	}

	_, err := handler.service.IssueReceipt(ctx, request)
	if err != nil {
		return fmt.Errorf("failed to issue receipt: %w", err)
	}

	return nil
}
