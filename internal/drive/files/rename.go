package files

import (
	"fmt"
	"io"

	"github.com/imzza/gdrive/internal/drive"
	gdrive "google.golang.org/api/drive/v3"
)

type RenameArgs struct {
	Out  io.Writer
	Id   string
	Name string
}

func Rename(drv *drive.Drive, args RenameArgs) error {
	f, err := drv.Service.Files.Get(args.Id).Fields("name").Do()
	if err != nil {
		return fmt.Errorf("Failed to get file: %s", err)
	}

	fmt.Fprintf(args.Out, "Renaming %s to %s\n", f.Name, args.Name)

	_, err = drv.Service.Files.Update(args.Id, &gdrive.File{Name: args.Name}).Fields("id,name").Do()
	if err != nil {
		return fmt.Errorf("Failed to rename file: %s", err)
	}

	return nil
}
