package drive

import (
	"context"
	"net/http"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

type Drive struct {
	service *drive.Service
}

func New(client *http.Client) (*Drive, error) {
	service, err := drive.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	return &Drive{service}, nil
}
