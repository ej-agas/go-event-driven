package event

import (
	"tickets/db"

	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
)

func RegisterEventHandlers(
	router *message.Router,
	config cqrs.EventProcessorConfig,
	spreadsheetsService SpreadsheetsAPI,
	receiptsService ReceiptsService,
	repository *db.TicketRepository,
) *cqrs.EventProcessor {
	eventProcessor, err := cqrs.NewEventProcessorWithConfig(router, config)
	if err != nil {
		panic(err)
	}

	err = eventProcessor.AddHandlers(
		NewAppendToTrackerHandler(spreadsheetsService),
		NewCancelTicketHandler(spreadsheetsService),
		NewIssueReceiptHandler(receiptsService),
		NewSaveToDatabaseHandler(repository),
	)
	if err != nil {
		panic(err)
	}

	return eventProcessor
}