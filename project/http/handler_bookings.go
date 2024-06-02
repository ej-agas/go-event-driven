package http

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"net/http"
	"tickets/entities"
)

func (h Handler) CreateBooking(c echo.Context) error {
	var booking entities.Booking

	if err := c.Bind(&booking); err != nil {
		return err
	}

	booking.ID = uuid.New()
	if err := h.bookingRepository.Create(c.Request().Context(), booking); err != nil {
		return fmt.Errorf("error: error creating booking: %v", err)
	}

	response := struct {
		BookingID string `json:"booking_id"`
	}{BookingID: booking.ID.String()}

	return c.JSON(http.StatusCreated, response)
}
