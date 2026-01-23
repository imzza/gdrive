package drive

import (
	"fmt"
	"io"

	"google.golang.org/api/drive/v3"
)

type RenameArgs struct {
	Out  io.Writer
	Id   string
	Name string
}

func (self *Drive) Rename(args RenameArgs) error {
	f, err := self.service.Files.Get(args.Id).Fields("name").Do()
	if err != nil {
		return fmt.Errorf("Failed to get file: %s", err)
	}

	fmt.Fprintf(args.Out, "Renaming %s to %s\n", f.Name, args.Name)

	_, err = self.service.Files.Update(args.Id, &drive.File{Name: args.Name}).Fields("id,name").Do()
	if err != nil {
		return fmt.Errorf("Failed to rename file: %s", err)
	}

	return nil
}
