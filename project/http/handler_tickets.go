package http

import (
	"fmt"
	"net/http"
	"tickets/entities"

	"github.com/labstack/echo/v4"
)

type ticketsStatusRequest struct {
	Tickets []ticketStatusRequest `json:"tickets"`
}

type ticketStatusRequest struct {
	TicketID      string         `json:"ticket_id"`
	Status        string         `json:"status"`
	Price         entities.Price `json:"price"`
	CustomerEmail string         `json:"customer_email"`
	BookingID     string         `json:"booking_id"`
}

func (h Handler) PostTicketsStatus(c echo.Context) error {
	idempotencyKey := c.Request().Header.Get("Idempotency-Key")
	var request ticketsStatusRequest
	err := c.Bind(&request)
	if err != nil {
		return err
	}

	for _, ticket := range request.Tickets {
		if ticket.Status == "confirmed" {
			event := entities.TicketBookingConfirmed{
				Header:        entities.NewEventHeaderWithIdempotencyKey(idempotencyKey),
				TicketID:      ticket.TicketID,
				Price:         ticket.Price,
				CustomerEmail: ticket.CustomerEmail,
			}

			if err := h.eventBus.Publish(c.Request().Context(), event); err != nil {
				return err
			}
		} else if ticket.Status == "canceled" {
			event := entities.TicketBookingCanceled{
				Header:        entities.NewEventHeaderWithIdempotencyKey(idempotencyKey),
				TicketID:      ticket.TicketID,
				CustomerEmail: ticket.CustomerEmail,
				Price:         ticket.Price,
			}

			if err := h.eventBus.Publish(c.Request().Context(), event); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("unknown ticket status: %s", ticket.Status)
		}
	}

	return c.NoContent(http.StatusOK)
}

func (h Handler) ListTickets(c echo.Context) error {
	tickets, err := h.ticketRepository.All(c.Request().Context())

	if err != nil {
		return fmt.Errorf("error fetching tickets: %w", err)
	}

	return c.JSON(http.StatusOK, tickets)
}
