package revision

import (
	"fmt"
	"github.com/imzza/gdrive/internal/drive"
	"github.com/imzza/gdrive/internal/drive/common"
	"io"
	"path/filepath"
	"time"
)

type DownloadRevisionArgs struct {
	Out        io.Writer
	Progress   io.Writer
	FileId     string
	RevisionId string
	Path       string
	Force      bool
	Stdout     bool
	Timeout    time.Duration
}

func DownloadRevision(drv *drive.Drive, args DownloadRevisionArgs) (err error) {
	getRev := drv.Service.Revisions.Get(args.FileId, args.RevisionId)

	rev, err := getRev.Fields("originalFilename").Do()
	if err != nil {
		return fmt.Errorf("Failed to get file: %s", err)
	}

	if rev.OriginalFilename == "" {
		return fmt.Errorf("Download is not supported for this file type")
	}

	// Get timeout reader wrapper and context
	timeoutReaderWrapper, ctx := common.GetTimeoutReaderWrapperContext(args.Timeout)

	res, err := getRev.Context(ctx).Download()
	if err != nil {
		if common.IsTimeoutError(err) {
			return fmt.Errorf("Failed to download file: timeout, no data was transferred for %v", args.Timeout)
		}
		return fmt.Errorf("Failed to download file: %s", err)
	}

	// Close body on function exit
	defer res.Body.Close()

	// Discard other output if file is written to stdout
	out := args.Out
	if args.Stdout {
		out = io.Discard
	}

	// Path to file
	fpath := filepath.Join(args.Path, rev.OriginalFilename)

	fmt.Fprintf(out, "Downloading %s -> %s\n", rev.OriginalFilename, fpath)

	bytes, rate, err := common.SaveFile(common.SaveFileArgs{
		Out:           args.Out,
		Body:          timeoutReaderWrapper(res.Body),
		ContentLength: res.ContentLength,
		Path:          fpath,
		Force:         args.Force,
		Stdout:        args.Stdout,
		Progress:      args.Progress,
	})

	if err != nil {
		return err
	}

	fmt.Fprintf(out, "Download complete, rate: %s/s, total size: %s\n", common.FormatSize(rate, false), common.FormatSize(bytes, false))
	return nil
}
