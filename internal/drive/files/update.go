package files

import (
	"fmt"
	"github.com/imzza/gdrive/internal/drive"
	"github.com/imzza/gdrive/internal/drive/common"
	gdrive "google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"
	"io"
	"mime"
	"path/filepath"
	"time"
)

type UpdateArgs struct {
	Out         io.Writer
	Progress    io.Writer
	Id          string
	Path        string
	Name        string
	Description string
	Parents     []string
	Mime        string
	Recursive   bool
	ChunkSize   int64
	Timeout     time.Duration
}

func Update(drv *drive.Drive, args UpdateArgs) error {
	srcFile, srcFileInfo, err := common.OpenFile(args.Path)
	if err != nil {
		return fmt.Errorf("Failed to open file: %s", err)
	}

	defer srcFile.Close()

	// Instantiate empty drive file
	dstFile := &gdrive.File{Description: args.Description}

	// Use provided file name or use filename
	if args.Name == "" {
		dstFile.Name = filepath.Base(srcFileInfo.Name())
	} else {
		dstFile.Name = args.Name
	}

	// Set provided mime type or get type based on file extension
	if args.Mime == "" {
		dstFile.MimeType = mime.TypeByExtension(filepath.Ext(dstFile.Name))
	} else {
		dstFile.MimeType = args.Mime
	}

	// Set parent folders
	dstFile.Parents = args.Parents

	// Chunk size option
	chunkSize := googleapi.ChunkSize(int(args.ChunkSize))

	// Wrap file in progress reader
	progressReader := common.GetProgressReader(srcFile, args.Progress, srcFileInfo.Size())

	// Wrap reader in timeout reader
	reader, ctx := common.GetTimeoutReaderContext(progressReader, args.Timeout)

	fmt.Fprintf(args.Out, "Uploading %s\n", args.Path)
	started := time.Now()

	f, err := drv.Service.Files.Update(args.Id, dstFile).Fields("id", "name", "size").Context(ctx).Media(reader, chunkSize).Do()
	if err != nil {
		if common.IsTimeoutError(err) {
			return fmt.Errorf("Failed to upload file: timeout, no data was transferred for %v", args.Timeout)
		}
		return fmt.Errorf("Failed to upload file: %s", err)
	}

	// Calculate average upload rate
	rate := common.CalcRate(f.Size, started, time.Now())

	fmt.Fprintf(args.Out, "Updated %s at %s/s, total %s\n", f.Id, common.FormatSize(rate, false), common.FormatSize(f.Size, false))
	return nil
}
