package message

import (
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

const brokenMessageID = "2beaf5bc-d5e4-4653-b075-2b36bbf28949"

func NewWatermillRouter(
	watermillLogger watermill.LoggerAdapter,
) *message.Router {
	router, err := message.NewRouter(message.RouterConfig{}, watermillLogger)
	if err != nil {
		panic(err)
	}

	useMiddlewares(router, watermillLogger)

	return router
}
