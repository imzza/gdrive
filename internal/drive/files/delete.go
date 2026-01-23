package files

import (
	"fmt"
	"io"

	"github.com/imzza/gdrive/internal/drive"
	"github.com/imzza/gdrive/internal/drive/common"
)

type DeleteArgs struct {
	Out       io.Writer
	Id        string
	Recursive bool
}

func Delete(drv *drive.Drive, args DeleteArgs) error {
	f, err := drv.Service.Files.Get(args.Id).Fields("name", "mimeType").Do()
	if err != nil {
		return fmt.Errorf("Failed to get file: %s", err)
	}

	if common.IsDir(f) && !args.Recursive {
		return fmt.Errorf("'%s' is a directory, use the 'recursive' flag to delete directories", f.Name)
	}

	err = drv.Service.Files.Delete(args.Id).Do()
	if err != nil {
		return fmt.Errorf("Failed to delete file: %s", err)
	}

	fmt.Fprintf(args.Out, "Deleted '%s'\n", f.Name)
	return nil
}

func deleteFile(drv *drive.Drive, fileId string) error {
	err := drv.Service.Files.Delete(fileId).Do()
	if err != nil {
		return fmt.Errorf("Failed to delete file: %s", err)
	}
	return nil
}
