package main

import (
	"database/sql"
	"fmt"
	watermillSQL "github.com/ThreeDotsLabs/watermill-sql/v2/pkg/sql"
	"github.com/ThreeDotsLabs/watermill/components/forwarder"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	_ "github.com/lib/pq"
)

var outboxTopic = "events_to_forward"

func PublishInTx(
	msg *message.Message,
	tx *sql.Tx,
	logger watermill.LoggerAdapter,
) error {
	var publisher message.Publisher
	var err error

	publisher, err = watermillSQL.NewPublisher(
		tx,
		watermillSQL.PublisherConfig{SchemaAdapter: watermillSQL.DefaultPostgreSQLSchema{}},
		logger,
	)

	if err != nil {
		return fmt.Errorf("error: could not create publisher: %w", err)
	}

	publisher = forwarder.NewPublisher(publisher, forwarder.PublisherConfig{
		ForwarderTopic: outboxTopic,
	})

	if err := publisher.Publish("ItemAddedToCart", msg); err != nil {
		return fmt.Errorf("error: could not publish message: %w", err)
	}

	return nil
}
