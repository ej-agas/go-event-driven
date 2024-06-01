package http

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"net/http"
	"tickets/entities"
	"time"
)

type createShowRequest struct {
	DeadNationID    uuid.UUID `json:"dead_nation_id"`
	NumberOfTickets int       `json:"number_of_tickets"`
	StartTime       time.Time `json:"start_time"`
	Title           string    `json:"title"`
	Venue           string    `json:"venue"`
}

func (h Handler) CreateShow(c echo.Context) error {
	var request createShowRequest
	if err := c.Bind(&request); err != nil {
		return err
	}

	show := entities.Show{
		ID:           uuid.New(),
		DeadNationID: request.DeadNationID,
		StartTime:    request.StartTime,
		Title:        request.Title,
		Venue:        request.Venue,
	}

	err := h.showRepository.Create(c.Request().Context(), show)

	if err != nil {
		return fmt.Errorf("error creating show: %w", err)
	}

	return c.JSON(http.StatusCreated, struct {
		ShowID string `json:"show_id"`
	}{ShowID: uuid.NewString()})
}
