package files

import (
	"fmt"
	"io"

	"github.com/imzza/gdrive/internal/drive"
	"github.com/imzza/gdrive/internal/drive/common"
	gdrive "google.golang.org/api/drive/v3"
)

type CopyArgs struct {
	Out      io.Writer
	Id       string
	FolderId string
}

func Copy(drv *drive.Drive, args CopyArgs) error {
	f, err := drv.Service.Files.Get(args.Id).Fields("name,mimeType").Do()
	if err != nil {
		return fmt.Errorf("Failed to get file: %s", err)
	}

	if common.IsDir(f) {
		return fmt.Errorf("Copy directories is not supported")
	}

	dest, err := drv.Service.Files.Get(args.FolderId).Fields("name,mimeType").Do()
	if err != nil {
		return fmt.Errorf("Failed to get destination folder: %s", err)
	}

	if !common.IsDir(dest) {
		return fmt.Errorf("Can only copy to a directory")
	}

	fmt.Fprintf(args.Out, "Copying '%s' to '%s'\n", f.Name, dest.Name)

	copyFile := &gdrive.File{
		Parents: []string{args.FolderId},
	}

	newFile, err := drv.Service.Files.Copy(args.Id, copyFile).Fields("id,name").SupportsAllDrives(true).Do()
	if err != nil {
		return fmt.Errorf("Failed to move file: %s", err)
	}

	fmt.Fprintf(args.Out, "Copied '%s' (id: %s)\n", newFile.Name, newFile.Id)
	return nil
}
