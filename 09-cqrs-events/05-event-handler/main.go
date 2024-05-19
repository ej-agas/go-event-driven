package main

import (
	"context"
	"fmt"

	"github.com/ThreeDotsLabs/watermill/components/cqrs"
)

type FollowRequestSent struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type EventsCounter interface {
	CountEvent() error
}

type EventHandler struct {
	counter EventsCounter
}

func NewFollowRequestSentHandler(counter EventsCounter) cqrs.EventHandler {
	return &EventHandler{
		counter: counter,
	}
}

func (h *EventHandler) HandlerName() string {
	return "FollowRequestSentHandler"
}

func (h *EventHandler) NewEvent() interface{} {
	return &FollowRequestSent{}
}

func (h *EventHandler) Handle(ctx context.Context, event any) error {
	if err := h.counter.CountEvent(); err != nil {
		return fmt.Errorf("error counting event: %w", err)
	}

	return nil
}
