package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ThreeDotsLabs/go-event-driven/common/clients"
	"github.com/ThreeDotsLabs/go-event-driven/common/log"
)

type FilesAPIClient struct {
	// we are not mocking this client: it's pointless to use interface here
	clients *clients.Clients
}

func NewFilesAPIClient(clients *clients.Clients) *FilesAPIClient {
	if clients == nil {
		panic("NewFilesAPIClient: clients is nil")
	}

	return &FilesAPIClient{clients: clients}
}

func (api *FilesAPIClient) Upload(ctx context.Context, name, contents string) error {
	res, err := api.clients.Files.PutFilesFileIdContentWithTextBodyWithResponse(ctx, name, contents)

	if err != nil {
		return fmt.Errorf("file api returned an error: %w", err)
	}

	if res.StatusCode() == http.StatusConflict {
		log.FromContext(ctx).Infof("file %s already exists", name)
		return nil
	}

	return nil
}
