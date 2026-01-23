package files

import (
	"fmt"
	"path/filepath"

	gdrive "google.golang.org/api/drive/v3"
)

func newPathfinder(service *gdrive.Service) *remotePathfinder {
	return &remotePathfinder{
		service: service.Files,
		files:   make(map[string]*gdrive.File),
	}
}

type remotePathfinder struct {
	service *gdrive.FilesService
	files   map[string]*gdrive.File
}

func (self *remotePathfinder) absPath(f *gdrive.File) (string, error) {
	name := f.Name

	if len(f.Parents) == 0 {
		return name, nil
	}

	var path []string

	for {
		parent, err := self.getParent(f.Parents[0])
		if err != nil {
			return "", err
		}

		// Stop when we find the root dir
		if len(parent.Parents) == 0 {
			break
		}

		path = append([]string{parent.Name}, path...)
		f = parent
	}

	path = append(path, name)
	return filepath.Join(path...), nil
}

func (self *remotePathfinder) getParent(id string) (*gdrive.File, error) {
	// Check cache
	if f, ok := self.files[id]; ok {
		return f, nil
	}

	// Fetch file from drive
	f, err := self.service.Get(id).Fields("id", "name", "parents").Do()
	if err != nil {
		return nil, fmt.Errorf("Failed to get file: %s", err)
	}

	// Save in cache
	self.files[f.Id] = f

	return f, nil
}
