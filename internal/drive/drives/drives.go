package drives

import (
	"context"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/imzza/gdrive/internal/drive"
	gdrive "google.golang.org/api/drive/v3"
)

type ListDrivesArgs struct {
	Out            io.Writer
	SkipHeader     bool
	FieldSeparator string
}

func ListDrives(drv *drive.Drive, args ListDrivesArgs) error {
	drives, err := listAllDrives(drv)
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

func listAllDrives(drv *drive.Drive) ([]*DriveInfo, error) {
	var drives []*DriveInfo

	err := drv.Service.Drives.List().Fields("nextPageToken", "drives(id,name)").Pages(context.TODO(), func(dl *gdrive.DriveList) error {
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
