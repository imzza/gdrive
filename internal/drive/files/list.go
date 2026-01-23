package files

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/imzza/gdrive/internal/drive"
	"github.com/imzza/gdrive/internal/drive/common"
	gdrive "google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"
)

type ListArgs struct {
	Out         io.Writer
	MaxFiles    int64
	NameWidth   int64
	Query       string
	SortOrder   string
	SkipHeader  bool
	SizeInBytes bool
	AbsPath     bool
}

func List(drv *drive.Drive, args ListArgs) (err error) {
	listArgs := drive.ListAllFilesArgs{
		Query:     args.Query,
		Fields:    []googleapi.Field{"nextPageToken", "files(id,name,md5Checksum,mimeType,size,createdTime,parents)"},
		SortOrder: args.SortOrder,
		MaxFiles:  args.MaxFiles,
	}
	files, err := drive.ListAllFiles(drv, listArgs)
	if err != nil {
		return fmt.Errorf("Failed to list files: %s", err)
	}

	pathfinder := newPathfinder(drv.Service)

	if args.AbsPath {
		// Replace name with absolute path
		for _, f := range files {
			f.Name, err = pathfinder.absPath(f)
			if err != nil {
				return err
			}
		}
	}

	PrintFileList(PrintFileListArgs{
		Out:         args.Out,
		Files:       files,
		NameWidth:   int(args.NameWidth),
		SkipHeader:  args.SkipHeader,
		SizeInBytes: args.SizeInBytes,
	})

	return
}

type PrintFileListArgs struct {
	Out         io.Writer
	Files       []*gdrive.File
	NameWidth   int
	SkipHeader  bool
	SizeInBytes bool
}

func PrintFileList(args PrintFileListArgs) {
	w := new(tabwriter.Writer)
	w.Init(args.Out, 0, 0, 3, ' ', 0)

	if !args.SkipHeader {
		fmt.Fprintln(w, "Id\tName\tType\tSize\tCreated")
	}

	for _, f := range args.Files {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			f.Id,
			common.TruncateString(f.Name, args.NameWidth),
			filetype(f),
			common.FormatSize(f.Size, args.SizeInBytes),
			common.FormatDatetime(f.CreatedTime),
		)
	}

	w.Flush()
}

func filetype(f *gdrive.File) string {
	if common.IsDir(f) {
		return "dir"
	} else if common.IsBinary(f) {
		return "bin"
	}
	return "doc"
}
