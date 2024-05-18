package message

import (
	"time"

	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/lithammer/shortuuid/v3"
	"github.com/sirupsen/logrus"
)

func useMiddlewares(router *message.Router, watermillLogger watermill.LoggerAdapter) {
	router.AddMiddleware(middleware.Recoverer)

	router.AddMiddleware(middleware.Retry{
		MaxRetries:      10,
		InitialInterval: time.Millisecond * 100,
		MaxInterval:     time.Second,
		Multiplier:      2,
		Logger:          watermillLogger,
	}.Middleware)

	router.AddMiddleware(correlationMiddleware)
	router.AddMiddleware(loggingMiddleware)
}

func correlationMiddleware(h message.HandlerFunc) message.HandlerFunc {
	return func(msg *message.Message) (events []*message.Message, err error) {
		ctx := msg.Context()

		reqCorrelationID := msg.Metadata.Get("correlation_id")
		if reqCorrelationID == "" {
			reqCorrelationID = shortuuid.New()
		}

		ctx = log.ToContext(ctx, logrus.WithFields(logrus.Fields{"correlation_id": reqCorrelationID}))
		ctx = log.ContextWithCorrelationID(ctx, reqCorrelationID)

		msg.SetContext(ctx)

		return h(msg)
	}
}

func loggingMiddleware(next message.HandlerFunc) message.HandlerFunc {
	return func(msg *message.Message) ([]*message.Message, error) {
		logger := log.FromContext(msg.Context())
		logger = logger.WithField("message_uuid", msg.UUID)

		msgs, err := next(msg)

		if err != nil {
			logger.WithError(err).Error("Message handling error")
		}

		logger.Info("Handling a message")

		return msgs, err
	}
}
