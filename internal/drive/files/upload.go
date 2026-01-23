package files

import (
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"time"

	"github.com/imzza/gdrive/internal/drive"
	"github.com/imzza/gdrive/internal/drive/common"
	"github.com/imzza/gdrive/internal/drive/permissions"
	syncdrive "github.com/imzza/gdrive/internal/drive/sync"
	gdrive "google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"
)

type UploadArgs struct {
	Out         io.Writer
	Progress    io.Writer
	Path        string
	Name        string
	Description string
	Parents     []string
	Mime        string
	Recursive   bool
	Share       bool
	Delete      bool
	ChunkSize   int64
	Timeout     time.Duration
}

func Upload(drv *drive.Drive, args UploadArgs) error {
	if args.ChunkSize > common.IntMax()-1 {
		return fmt.Errorf("Chunk size is to big, max chunk size for this computer is %d", common.IntMax()-1)
	}

	// Ensure that none of the parents are sync dirs
	for _, parent := range args.Parents {
		isSyncDir, err := syncdrive.IsSyncFile(drv, parent)
		if err != nil {
			return err
		}

		if isSyncDir {
			return fmt.Errorf("%s is a sync directory, use 'sync upload' instead", parent)
		}
	}

	if args.Recursive {
		return uploadRecursive(drv, args)
	}

	info, err := os.Stat(args.Path)
	if err != nil {
		return fmt.Errorf("Failed stat file: %s", err)
	}

	if info.IsDir() {
		return fmt.Errorf("'%s' is a directory, use --recursive to upload directories", info.Name())
	}

	f, rate, err := uploadFile(drv, args)
	if err != nil {
		return err
	}
	fmt.Fprintf(args.Out, "Uploaded %s at %s/s, total %s\n", f.Id, common.FormatSize(rate, false), common.FormatSize(f.Size, false))

	if args.Share {
		err = permissions.ShareAnyoneReader(drv, f.Id)
		if err != nil {
			return err
		}

		fmt.Fprintf(args.Out, "File is readable by anyone at %s\n", f.WebContentLink)
	}

	if args.Delete {
		err = os.Remove(args.Path)
		if err != nil {
			return fmt.Errorf("Failed to delete file: %s", err)
		}
		fmt.Fprintf(args.Out, "Removed %s\n", args.Path)
	}

	return nil
}

func uploadRecursive(drv *drive.Drive, args UploadArgs) error {
	info, err := os.Stat(args.Path)
	if err != nil {
		return fmt.Errorf("Failed stat file: %s", err)
	}

	if info.IsDir() {
		args.Name = ""
		return uploadDirectory(drv, args)
	} else if info.Mode().IsRegular() {
		_, _, err := uploadFile(drv, args)
		return err
	}

	return nil
}

func uploadDirectory(drv *drive.Drive, args UploadArgs) error {
	srcFile, srcFileInfo, err := common.OpenFile(args.Path)
	if err != nil {
		return err
	}

	// Close file on function exit
	defer srcFile.Close()

	fmt.Fprintf(args.Out, "Creating directory %s\n", srcFileInfo.Name())
	// Make directory on drive
	f, err := mkdir(drv, MkdirArgs{
		Out:         args.Out,
		Name:        srcFileInfo.Name(),
		Parents:     args.Parents,
		Description: args.Description,
	})
	if err != nil {
		return err
	}

	// Read files from directory
	names, err := srcFile.Readdirnames(0)
	if err != nil && err != io.EOF {
		return fmt.Errorf("Failed reading directory: %s", err)
	}

	for _, name := range names {
		// Copy args and set new path and parents
		newArgs := args
		newArgs.Path = filepath.Join(args.Path, name)
		newArgs.Parents = []string{f.Id}
		newArgs.Description = ""

		// Upload
		err = uploadRecursive(drv, newArgs)
		if err != nil {
			return err
		}
	}

	return nil
}

func uploadFile(drv *drive.Drive, args UploadArgs) (*gdrive.File, int64, error) {
	srcFile, srcFileInfo, err := common.OpenFile(args.Path)
	if err != nil {
		return nil, 0, err
	}

	// Close file on function exit
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

	f, err := drv.Service.Files.Create(dstFile).Fields("id", "name", "size", "md5Checksum", "webContentLink").Context(ctx).Media(reader, chunkSize).Do()
	if err != nil {
		if common.IsTimeoutError(err) {
			return nil, 0, fmt.Errorf("Failed to upload file: timeout, no data was transferred for %v", args.Timeout)
		}
		return nil, 0, fmt.Errorf("Failed to upload file: %s", err)
	}

	// Calculate average upload rate
	rate := common.CalcRate(f.Size, started, time.Now())

	return f, rate, nil
}

type UploadStreamArgs struct {
	Out         io.Writer
	In          io.Reader
	Name        string
	Description string
	Parents     []string
	Mime        string
	Share       bool
	ChunkSize   int64
	Progress    io.Writer
	Timeout     time.Duration
}

func UploadStream(drv *drive.Drive, args UploadStreamArgs) error {
	if args.ChunkSize > common.IntMax()-1 {
		return fmt.Errorf("Chunk size is to big, max chunk size for this computer is %d", common.IntMax()-1)
	}

	// Instantiate empty drive file
	dstFile := &gdrive.File{Name: args.Name, Description: args.Description}

	// Set mime type if provided
	if args.Mime != "" {
		dstFile.MimeType = args.Mime
	}

	// Set parent folders
	dstFile.Parents = args.Parents

	// Chunk size option
	chunkSize := googleapi.ChunkSize(int(args.ChunkSize))

	// Wrap file in progress reader
	progressReader := common.GetProgressReader(args.In, args.Progress, 0)

	// Wrap reader in timeout reader
	reader, ctx := common.GetTimeoutReaderContext(progressReader, args.Timeout)

	fmt.Fprintf(args.Out, "Uploading %s\n", dstFile.Name)
	started := time.Now()

	f, err := drv.Service.Files.Create(dstFile).Fields("id", "name", "size", "webContentLink").Context(ctx).Media(reader, chunkSize).Do()
	if err != nil {
		if common.IsTimeoutError(err) {
			return fmt.Errorf("Failed to upload file: timeout, no data was transferred for %v", args.Timeout)
		}
		return fmt.Errorf("Failed to upload file: %s", err)
	}

	// Calculate average upload rate
	rate := common.CalcRate(f.Size, started, time.Now())

	fmt.Fprintf(args.Out, "Uploaded %s at %s/s, total %s\n", f.Id, common.FormatSize(rate, false), common.FormatSize(f.Size, false))
	if args.Share {
		err = permissions.ShareAnyoneReader(drv, f.Id)
		if err != nil {
			return err
		}

		fmt.Fprintf(args.Out, "File is readable by anyone at %s\n", f.WebContentLink)
	}
	return nil
}
