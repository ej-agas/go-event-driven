package main

import (
	"os"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/redis/go-redis/v9"
)

func main() {
	redisClient := redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_ADDR"),
	})

	logger := watermill.NewStdLogger(false, false)

	publisher, err := redisstream.NewPublisher(redisstream.PublisherConfig{
		Client: redisClient,
	}, logger)

	if err != nil {
		panic(err)
	}

	msg1 := message.NewMessage(watermill.NewUUID(), []byte("50"))
	if err := publisher.Publish("progress", msg1); err != nil {
		panic(err)
	}

	msg2 := message.NewMessage(watermill.NewUUID(), []byte("100"))
	if err := publisher.Publish("progress", msg2); err != nil {
		panic(err)
	}
}
