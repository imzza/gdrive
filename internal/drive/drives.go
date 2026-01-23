package drive

import (
	"context"
	"fmt"
	"io"
	"text/tabwriter"

	"google.golang.org/api/drive/v3"
)

type ListDrivesArgs struct {
	Out            io.Writer
	SkipHeader     bool
	FieldSeparator string
}

func (self *Drive) ListDrives(args ListDrivesArgs) error {
	drives, err := self.listAllDrives()
	if err != nil {
		return fmt.Errorf("Failed to list drives: %s", err)
	}

	if args.FieldSeparator == "\t" {
		w := new(tabwriter.Writer)
		w.Init(args.Out, 0, 0, 3, ' ', 0)

		if !args.SkipHeader {
			fmt.Fprintln(w, "Id\tName")
		}

		for _, d := range drives {
			fmt.Fprintf(w, "%s\t%s\n", d.Id, d.Name)
		}

		w.Flush()
		return nil
	}

	sep := args.FieldSeparator
	if sep == "" {
		sep = "\t"
	}

	if !args.SkipHeader {
		fmt.Fprintf(args.Out, "Id%sName\n", sep)
	}

	for _, d := range drives {
		fmt.Fprintf(args.Out, "%s%s%s\n", d.Id, sep, d.Name)
	}

	return nil
}

func (self *Drive) listAllDrives() ([]*DriveInfo, error) {
	var drives []*DriveInfo

	err := self.service.Drives.List().Fields("nextPageToken", "drives(id,name)").Pages(context.TODO(), func(dl *drive.DriveList) error {
		for _, d := range dl.Drives {
			drives = append(drives, &DriveInfo{
				Id:   d.Id,
				Name: d.Name,
			})
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return drives, nil
}

type DriveInfo struct {
	Id   string
	Name string
}
