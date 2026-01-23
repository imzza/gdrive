package files

import (
	"fmt"
	"io"

	"github.com/imzza/gdrive/internal/drive"
	"github.com/imzza/gdrive/internal/drive/common"
	gdrive "google.golang.org/api/drive/v3"
)

type InfoArgs struct {
	Out         io.Writer
	Id          string
	SizeInBytes bool
}

func Info(drv *drive.Drive, args InfoArgs) error {
	f, err := drv.Service.Files.Get(args.Id).Fields("id", "name", "size", "createdTime", "modifiedTime", "md5Checksum", "mimeType", "parents", "shared", "description", "webContentLink", "webViewLink").Do()
	if err != nil {
		return fmt.Errorf("Failed to get file: %s", err)
	}

	pathfinder := newPathfinder(drv.Service)
	absPath, err := pathfinder.absPath(f)
	if err != nil {
		return err
	}

	PrintFileInfo(PrintFileInfoArgs{
		Out:         args.Out,
		File:        f,
		Path:        absPath,
		SizeInBytes: args.SizeInBytes,
	})

	return nil
}

type PrintFileInfoArgs struct {
	Out         io.Writer
	File        *gdrive.File
	Path        string
	SizeInBytes bool
}

type kv struct {
	key   string
	value string
}

func PrintFileInfo(args PrintFileInfoArgs) {
	f := args.File

	items := []kv{
		{"Id", f.Id},
		{"Name", f.Name},
		{"Path", args.Path},
		{"Description", f.Description},
		{"Mime", f.MimeType},
		{"Size", common.FormatSize(f.Size, args.SizeInBytes)},
		{"Created", common.FormatDatetime(f.CreatedTime)},
		{"Modified", common.FormatDatetime(f.ModifiedTime)},
		{"Md5sum", f.Md5Checksum},
		{"Shared", common.FormatBool(f.Shared)},
		{"Parents", common.FormatList(f.Parents)},
		{"ViewUrl", f.WebViewLink},
		{"DownloadUrl", f.WebContentLink},
	}

	for _, item := range items {
		if item.value != "" {
			fmt.Fprintf(args.Out, "%s: %s\n", item.key, item.value)
		}
	}
}
