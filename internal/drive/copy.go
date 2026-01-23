package drive

import (
	"fmt"
	"io"

	"google.golang.org/api/drive/v3"
)

type CopyArgs struct {
	Out      io.Writer
	Id       string
	FolderId string
}

func (self *Drive) Copy(args CopyArgs) error {
	f, err := self.service.Files.Get(args.Id).Fields("name,mimeType").Do()
	if err != nil {
		return fmt.Errorf("Failed to get file: %s", err)
	}

	if isDir(f) {
		return fmt.Errorf("Copy directories is not supported")
	}

	dest, err := self.service.Files.Get(args.FolderId).Fields("name,mimeType").Do()
	if err != nil {
		return fmt.Errorf("Failed to get destination folder: %s", err)
	}

	if !isDir(dest) {
		return fmt.Errorf("Can only copy to a directory")
	}

	fmt.Fprintf(args.Out, "Copying '%s' to '%s'\n", f.Name, dest.Name)

	copyFile := &drive.File{
		Parents: []string{args.FolderId},
	}

	newFile, err := self.service.Files.Copy(args.Id, copyFile).Fields("id,name").SupportsAllDrives(true).Do()
	if err != nil {
		return fmt.Errorf("Failed to move file: %s", err)
	}

	fmt.Fprintf(args.Out, "Copied '%s' (id: %s)\n", newFile.Name, newFile.Id)
	return nil
}
