package files

import (
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/imzza/gdrive/internal/drive"
	"github.com/imzza/gdrive/internal/drive/common"
	gdrive "google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"
)

type DownloadArgs struct {
	Out       io.Writer
	Progress  io.Writer
	Id        string
	Path      string
	Force     bool
	Skip      bool
	Recursive bool
	Delete    bool
	Stdout    bool
	Timeout   time.Duration
}

func Download(drv *drive.Drive, args DownloadArgs) error {
	if args.Recursive {
		return downloadRecursive(drv, args)
	}

	f, err := drv.Service.Files.Get(args.Id).Fields("id", "name", "size", "mimeType", "md5Checksum").Do()
	if err != nil {
		return fmt.Errorf("Failed to get file: %s", err)
	}

	if common.IsDir(f) {
		return fmt.Errorf("'%s' is a directory, use --recursive to download directories", f.Name)
	}

	if !common.IsBinary(f) {
		return fmt.Errorf("'%s' is a google document and must be exported, see the export command", f.Name)
	}

	bytes, rate, err := downloadBinary(drv, f, args)
	if err != nil {
		return err
	}

	if !args.Stdout {
		fmt.Fprintf(args.Out, "Downloaded %s at %s/s, total %s\n", f.Id, common.FormatSize(rate, false), common.FormatSize(bytes, false))
	}

	if args.Delete {
		err = deleteFile(drv, args.Id)
		if err != nil {
			return fmt.Errorf("Failed to delete file: %s", err)
		}

		if !args.Stdout {
			fmt.Fprintf(args.Out, "Removed %s\n", args.Id)
		}
	}
	return err
}

type DownloadQueryArgs struct {
	Out       io.Writer
	Progress  io.Writer
	Query     string
	Path      string
	Force     bool
	Skip      bool
	Recursive bool
}

func DownloadQuery(drv *drive.Drive, args DownloadQueryArgs) error {
	listArgs := drive.ListAllFilesArgs{
		Query:  args.Query,
		Fields: []googleapi.Field{"nextPageToken", "files(id,name,mimeType,size,md5Checksum)"},
	}
	files, err := drive.ListAllFiles(drv, listArgs)
	if err != nil {
		return fmt.Errorf("Failed to list files: %s", err)
	}

	downloadArgs := DownloadArgs{
		Out:      args.Out,
		Progress: args.Progress,
		Path:     args.Path,
		Force:    args.Force,
		Skip:     args.Skip,
	}

	for _, f := range files {
		if common.IsDir(f) && args.Recursive {
			err = downloadDirectory(drv, f, downloadArgs)
		} else if common.IsBinary(f) {
			_, _, err = downloadBinary(drv, f, downloadArgs)
		}

		if err != nil {
			return err
		}
	}

	return nil
}

func downloadRecursive(drv *drive.Drive, args DownloadArgs) error {
	f, err := drv.Service.Files.Get(args.Id).Fields("id", "name", "size", "mimeType", "md5Checksum").Do()
	if err != nil {
		return fmt.Errorf("Failed to get file: %s", err)
	}

	if common.IsDir(f) {
		return downloadDirectory(drv, f, args)
	} else if common.IsBinary(f) {
		_, _, err = downloadBinary(drv, f, args)
		return err
	}

	return nil
}

func downloadBinary(drv *drive.Drive, f *gdrive.File, args DownloadArgs) (int64, int64, error) {
	// Get timeout reader wrapper and context
	timeoutReaderWrapper, ctx := common.GetTimeoutReaderWrapperContext(args.Timeout)

	res, err := drv.Service.Files.Get(f.Id).Context(ctx).Download()
	if err != nil {
		if common.IsTimeoutError(err) {
			return 0, 0, fmt.Errorf("Failed to download file: timeout, no data was transferred for %v", args.Timeout)
		}
		return 0, 0, fmt.Errorf("Failed to download file: %s", err)
	}

	// Close body on function exit
	defer res.Body.Close()

	// Path to file
	fpath := filepath.Join(args.Path, f.Name)

	if !args.Stdout {
		fmt.Fprintf(args.Out, "Downloading %s -> %s\n", f.Name, fpath)
	}

	return common.SaveFile(common.SaveFileArgs{
		Out:           args.Out,
		Body:          timeoutReaderWrapper(res.Body),
		ContentLength: res.ContentLength,
		Path:          fpath,
		Force:         args.Force,
		Skip:          args.Skip,
		Stdout:        args.Stdout,
		Progress:      args.Progress,
	})
}

func downloadDirectory(drv *drive.Drive, parent *gdrive.File, args DownloadArgs) error {
	listArgs := drive.ListAllFilesArgs{
		Query:  fmt.Sprintf("'%s' in parents", parent.Id),
		Fields: []googleapi.Field{"nextPageToken", "files(id,name)"},
	}
	files, err := drive.ListAllFiles(drv, listArgs)
	if err != nil {
		return fmt.Errorf("Failed listing files: %s", err)
	}

	newPath := filepath.Join(args.Path, parent.Name)

	for _, f := range files {
		// Copy args and update changed fields
		newArgs := args
		newArgs.Path = newPath
		newArgs.Id = f.Id
		newArgs.Stdout = false

		err = downloadRecursive(drv, newArgs)
		if err != nil {
			return err
		}
	}

	return nil
}
