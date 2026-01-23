package drive

import (
	"fmt"
	"io"
	"strings"

	"google.golang.org/api/drive/v3"
)

type MoveArgs struct {
	Out      io.Writer
	Id       string
	FolderId string
}

func (self *Drive) Move(args MoveArgs) error {
	f, err := self.service.Files.Get(args.Id).Fields("name,parents").Do()
	if err != nil {
		return fmt.Errorf("Failed to get file: %s", err)
	}

	oldParentId, err := singleParentId(f.Parents)
	if err != nil {
		return err
	}

	oldParent, err := self.service.Files.Get(oldParentId).Fields("name").Do()
	if err != nil {
		return fmt.Errorf("Failed to get old parent '%s': %s", oldParentId, err)
	}

	newParent, err := self.service.Files.Get(args.FolderId).Fields("name,mimeType").Do()
	if err != nil {
		return fmt.Errorf("Failed to get new parent: %s", err)
	}

	if !isDir(newParent) {
		return fmt.Errorf("New parent is not a directory")
	}

	fmt.Fprintf(args.Out, "Moving '%s' from '%s' to '%s'\n", f.Name, oldParent.Name, newParent.Name)

	_, err = self.service.Files.Update(args.Id, &drive.File{}).
		AddParents(args.FolderId).
		RemoveParents(oldParentId).
		SupportsAllDrives(true).
		Do()
	if err != nil {
		return fmt.Errorf("Failed to move file: %s", err)
	}

	return nil
}

func singleParentId(parents []string) (string, error) {
	if len(parents) == 0 {
		return "", fmt.Errorf("File has no parents")
	}
	if len(parents) > 1 {
		return "", fmt.Errorf("Can't move file with multiple parents")
	}

	if strings.TrimSpace(parents[0]) == "" {
		return "", fmt.Errorf("File has no parents")
	}

	return parents[0], nil
}
