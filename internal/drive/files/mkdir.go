package files

import (
	"fmt"
	"io"

	"github.com/imzza/gdrive/internal/drive"
	"github.com/imzza/gdrive/internal/drive/common"
	gdrive "google.golang.org/api/drive/v3"
)

type MkdirArgs struct {
	Out         io.Writer
	Name        string
	Description string
	Parents     []string
}

func Mkdir(drv *drive.Drive, args MkdirArgs) error {
	f, err := mkdir(drv, args)
	if err != nil {
		return err
	}
	fmt.Fprintf(args.Out, "Directory %s created\n", f.Id)
	return nil
}

func mkdir(drv *drive.Drive, args MkdirArgs) (*gdrive.File, error) {
	dstFile := &gdrive.File{
		Name:        args.Name,
		Description: args.Description,
		MimeType:    common.DirectoryMimeType,
	}

	// Set parent folders
	dstFile.Parents = args.Parents

	// Create directory
	f, err := drv.Service.Files.Create(dstFile).Do()
	if err != nil {
		return nil, fmt.Errorf("Failed to create directory: %s", err)
	}

	return f, nil
}
