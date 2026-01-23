package drive

import (
	"google.golang.org/api/drive/v3"
	"net/http"
)

type Drive struct {
	Service *drive.Service
}

func New(client *http.Client) (*Drive, error) {
	service, err := drive.New(client)
	if err != nil {
		return nil, err
	}

	return &Drive{Service: service}, nil
}
