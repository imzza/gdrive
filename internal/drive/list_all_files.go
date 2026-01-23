package drive

import (
	"context"
	"fmt"

	"github.com/imzza/gdrive/internal/drive/common"
	gdrive "google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"
)

type ListAllFilesArgs struct {
	Query     string
	Fields    []googleapi.Field
	SortOrder string
	MaxFiles  int64
}

func ListAllFiles(drv *Drive, args ListAllFilesArgs) ([]*gdrive.File, error) {
	var files []*gdrive.File

	var pageSize int64
	if args.MaxFiles > 0 && args.MaxFiles < 1000 {
		pageSize = args.MaxFiles
	} else {
		pageSize = 1000
	}

	controlledStop := fmt.Errorf("Controlled stop")

	err := drv.Service.Files.List().Q(args.Query).Fields(args.Fields...).OrderBy(args.SortOrder).PageSize(pageSize).Pages(context.TODO(), func(fl *gdrive.FileList) error {
		files = append(files, fl.Files...)

		// Stop when we have all the files we need
		if args.MaxFiles > 0 && len(files) >= int(args.MaxFiles) {
			return controlledStop
		}

		return nil
	})

	if err != nil && err != controlledStop {
		return nil, err
	}

	if args.MaxFiles > 0 {
		n := common.Min(len(files), int(args.MaxFiles))
		return files[:n], nil
	}

	return files, nil
}
