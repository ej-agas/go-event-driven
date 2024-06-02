package main

import (
	"context"
	"fmt"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"github.com/ThreeDotsLabs/watermill-sql/v2/pkg/sql"
	"github.com/ThreeDotsLabs/watermill/components/forwarder"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

func RunForwarder(
	db *sqlx.DB,
	rdb *redis.Client,
	outboxTopic string,
	logger watermill.LoggerAdapter,
) error {
	subscriber, err := sql.NewSubscriber(
		db,
		sql.SubscriberConfig{
			SchemaAdapter:  sql.DefaultPostgreSQLSchema{},
			OffsetsAdapter: sql.DefaultPostgreSQLOffsetsAdapter{},
		},
		logger,
	)

	if err != nil {
		return fmt.Errorf("error: failed to create subscriber: %w", err)
	}

	if err := subscriber.SubscribeInitialize(outboxTopic); err != nil {
		return fmt.Errorf("error: failed to initialize subscriber: %w", err)
	}

	publisher, err := redisstream.NewPublisher(
		redisstream.PublisherConfig{Client: rdb},
		logger,
	)

	fwd, err := forwarder.NewForwarder(subscriber, publisher, logger, forwarder.Config{
		ForwarderTopic: outboxTopic,
	})

	go func() {
		err := fwd.Run(context.Background())
		if err != nil {
			panic(fmt.Errorf("error: failed to run forwarder: %w", err))
		}
	}()
	<-fwd.Running()

	return nil
}
