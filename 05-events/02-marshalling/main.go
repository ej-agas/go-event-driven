package main

import (
	"context"
	"encoding/json"
	"os"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/redis/go-redis/v9"
)

type PaymentCompleted struct {
	PaymentID   string `json:"payment_id"`
	OrderID     string `json:"order_id"`
	CompletedAt string `json:"completed_at"`
}

type OrderConfirmed struct {
	OrderID     string `json:"order_id"`
	ConfirmedAt string `json:"confirmed_at"`
}

func main() {
	logger := watermill.NewStdLogger(false, false)

	router, err := message.NewRouter(message.RouterConfig{}, logger)
	if err != nil {
		panic(err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_ADDR"),
	})

	sub, err := redisstream.NewSubscriber(redisstream.SubscriberConfig{
		Client: rdb,
	}, logger)
	if err != nil {
		panic(err)
	}

	pub, err := redisstream.NewPublisher(redisstream.PublisherConfig{
		Client: rdb,
	}, logger)
	if err != nil {
		panic(err)
	}

	router.AddHandler(
		"handle-payment-completed",
		"payment-completed",
		sub,
		"order-confirmed",
		pub,
		handlePaymentCompleted,
	)

	err = router.Run(context.Background())
	if err != nil {
		panic(err)
	}
}

func handlePaymentCompleted(msg *message.Message) ([]*message.Message, error) {
	var event PaymentCompleted

	if err := json.Unmarshal(msg.Payload, &event); err != nil {
		return nil, err
	}

	newEvent := OrderConfirmed{
		OrderID:     event.OrderID,
		ConfirmedAt: event.CompletedAt,
	}

	payload, err := json.Marshal(newEvent)

	if err != nil {
		return nil, err
	}

	newMsg := message.NewMessage(watermill.NewUUID(), payload)

	return []*message.Message{newMsg}, nil
}
