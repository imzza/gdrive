package sync

import (
	"fmt"
	"io"
	"sort"
	"text/tabwriter"

	"github.com/imzza/gdrive/internal/drive"
	"github.com/imzza/gdrive/internal/drive/common"
	gdrive "google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"
)

type ListSyncArgs struct {
	Out        io.Writer
	SkipHeader bool
}

func ListSync(drv *drive.Drive, args ListSyncArgs) error {
	listArgs := drive.ListAllFilesArgs{
		Query:  "appProperties has {key='syncRoot' and value='true'}",
		Fields: []googleapi.Field{"nextPageToken", "files(id,name,mimeType,createdTime)"},
	}
	files, err := drive.ListAllFiles(drv, listArgs)
	if err != nil {
		return err
	}
	printSyncDirectories(files, args)
	return nil
}

type ListRecursiveSyncArgs struct {
	Out         io.Writer
	RootId      string
	SkipHeader  bool
	PathWidth   int64
	SizeInBytes bool
	SortOrder   string
}

func ListRecursiveSync(drv *drive.Drive, args ListRecursiveSyncArgs) error {
	rootDir, err := getSyncRoot(drv, args.RootId)
	if err != nil {
		return err
	}

	files, err := prepareRemoteFiles(drv, rootDir, args.SortOrder)
	if err != nil {
		return err
	}

	printSyncDirContent(files, args)
	return nil
}

func printSyncDirectories(files []*gdrive.File, args ListSyncArgs) {
	w := new(tabwriter.Writer)
	w.Init(args.Out, 0, 0, 3, ' ', 0)

	if !args.SkipHeader {
		fmt.Fprintln(w, "Id\tName\tCreated")
	}

	for _, f := range files {
		fmt.Fprintf(w, "%s\t%s\t%s\n",
			f.Id,
			f.Name,
			common.FormatDatetime(f.CreatedTime),
		)
	}

	w.Flush()
}

func printSyncDirContent(files []*RemoteFile, args ListRecursiveSyncArgs) {
	if args.SortOrder == "" {
		// Sort files by path
		sort.Sort(byRemotePath(files))
	}

	w := new(tabwriter.Writer)
	w.Init(args.Out, 0, 0, 3, ' ', 0)

	if !args.SkipHeader {
		fmt.Fprintln(w, "Id\tPath\tType\tSize\tModified")
	}

	for _, rf := range files {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			rf.file.Id,
			common.TruncateString(rf.relPath, int(args.PathWidth)),
			filetype(rf.file),
			common.FormatSize(rf.file.Size, args.SizeInBytes),
			common.FormatDatetime(rf.file.ModifiedTime),
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
