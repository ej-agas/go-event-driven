package api

import (
	"context"
	"fmt"
	"sync"

	"github.com/ThreeDotsLabs/go-event-driven/common/log"
)

type FilesAPIClientMock struct {
	sync.Mutex
	Files map[string]string
}

func (api *FilesAPIClientMock) Upload(ctx context.Context, name, contents string) error {
	api.Lock()
	defer api.Unlock()

	if api.Files == nil {
		api.Files = make(map[string]string)
	}

	if _, alreadyExists := api.Files[name]; alreadyExists {
		log.FromContext(ctx).Infof("file %s already exists", name)
		return nil
	}

	api.Files[name] = contents

	return nil
}

func (api *FilesAPIClientMock) Download(ctx context.Context, name string) (string, error) {
	api.Lock()
	defer api.Unlock()

	if api.Files == nil {
		api.Files = make(map[string]string)
	}

	fileContent, ok := api.Files[name]
	if !ok {
		return "", fmt.Errorf("file %s not found", name)
	}

	return fileContent, nil
}
