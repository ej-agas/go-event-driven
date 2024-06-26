package event

import (
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
)

func RegisterEventHandlers(
	router *message.Router,
	config cqrs.EventProcessorConfig,
	spreadsheetsService SpreadsheetsAPI,
	receiptsService ReceiptsService,
	repository TicketsRepository,
	filesService FilesAPI,
	eventBus *cqrs.EventBus,
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
		NewDeleteCanceledTicketsHandler(repository),
		NewSaveToFileHandler(filesService, eventBus),
	)
	if err != nil {
		panic(err)
	}

	return eventProcessor
}
