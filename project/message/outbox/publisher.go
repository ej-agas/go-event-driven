package outbox

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	watermillSQL "github.com/ThreeDotsLabs/watermill-sql/v2/pkg/sql"
	"github.com/ThreeDotsLabs/watermill/components/forwarder"
	"github.com/ThreeDotsLabs/watermill/message"
)

func NewPublisherForDb(ctx context.Context, db *sql.Tx) (message.Publisher, error) {
	var publisher message.Publisher

	logger := log.NewWatermill(log.FromContext(ctx))

	publisher, err := watermillSQL.NewPublisher(
		db,
		watermillSQL.PublisherConfig{
			SchemaAdapter: watermillSQL.DefaultPostgreSQLSchema{},
		},
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create outbox publisher: %w", err)
	}
	publisher = log.CorrelationPublisherDecorator{Publisher: publisher}

	publisher = forwarder.NewPublisher(publisher, forwarder.PublisherConfig{
		ForwarderTopic: outboxTopic,
	})
	publisher = log.CorrelationPublisherDecorator{Publisher: publisher}

	return publisher, nil
}
