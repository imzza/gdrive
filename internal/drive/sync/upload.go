package sync

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/imzza/gdrive/internal/drive"
	"github.com/imzza/gdrive/internal/drive/common"
	gdrive "google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"
)

type UploadSyncArgs struct {
	Out              io.Writer
	Progress         io.Writer
	Path             string
	RootId           string
	DryRun           bool
	DeleteExtraneous bool
	ChunkSize        int64
	Timeout          time.Duration
	Resolution       ConflictResolution
	Comparer         FileComparer
}

func UploadSync(drv *drive.Drive, args UploadSyncArgs) error {
	if args.ChunkSize > common.IntMax()-1 {
		return fmt.Errorf("Chunk size is to big, max chunk size for this computer is %d", common.IntMax()-1)
	}

	fmt.Fprintln(args.Out, "Starting sync...")
	started := time.Now()

	// Create root directory if it does not exist
	rootDir, err := prepareSyncRoot(drv, args)
	if err != nil {
		return err
	}

	fmt.Fprintln(args.Out, "Collecting local and remote file information...")
	files, err := prepareSyncFiles(drv, args.Path, rootDir, args.Comparer)
	if err != nil {
		return err
	}

	// Find missing and changed files
	changedFiles := files.filterChangedLocalFiles()
	missingFiles := files.filterMissingRemoteFiles()

	fmt.Fprintf(args.Out, "Found %d local files and %d remote files\n", len(files.local), len(files.remote))

	// Ensure that there is enough free space on drive
	if ok, msg := checkRemoteFreeSpace(drv, missingFiles, changedFiles); !ok {
		return fmt.Errorf("%s", msg)
	}

	// Ensure that we don't overwrite any remote changes
	if args.Resolution == NoResolution {
		err = ensureNoRemoteModifications(changedFiles)
		if err != nil {
			return fmt.Errorf("Conflict detected!\nThe following files have changed and the remote file are newer than it's local counterpart:\n\n%s\nNo conflict resolution was given, aborting...", err)
		}
	}

	// Create missing directories
	files, err = createMissingRemoteDirs(drv, files, args)
	if err != nil {
		return err
	}

	// Upload missing files
	err = uploadMissingFiles(drv, missingFiles, files, args)
	if err != nil {
		return err
	}

	// Update modified files
	err = updateChangedFiles(drv, changedFiles, rootDir, args)
	if err != nil {
		return err
	}

	// Delete extraneous files on drive
	if args.DeleteExtraneous {
		err = deleteExtraneousRemoteFiles(drv, files, args)
		if err != nil {
			return err
		}
	}
	fmt.Fprintf(args.Out, "Sync finished in %s\n", time.Since(started))

	return nil
}

func prepareSyncRoot(drv *drive.Drive, args UploadSyncArgs) (*gdrive.File, error) {
	fields := []googleapi.Field{"id", "name", "mimeType", "appProperties"}
	f, err := drv.Service.Files.Get(args.RootId).Fields(fields...).Do()
	if err != nil {
		return nil, fmt.Errorf("Failed to find root dir: %s", err)
	}

	// Ensure file is a directory
	if !common.IsDir(f) {
		return nil, fmt.Errorf("Provided root id is not a directory")
	}

	// Return directory if syncRoot property is already set
	if _, ok := f.AppProperties["syncRoot"]; ok {
		return f, nil
	}

	// This is the first time this directory have been used for sync
	// Check if the directory is empty
	isEmpty, err := dirIsEmpty(drv, f.Id)
	if err != nil {
		return nil, fmt.Errorf("Failed to check if root dir is empty: %s", err)
	}

	// Ensure that the directory is empty
	if !isEmpty {
		return nil, fmt.Errorf("Root directory is not empty, the initial sync requires an empty directory")
	}

	// Update directory with syncRoot property
	dstFile := &gdrive.File{
		AppProperties: map[string]string{"sync": "true", "syncRoot": "true"},
	}

	f, err = drv.Service.Files.Update(f.Id, dstFile).Fields(fields...).Do()
	if err != nil {
		return nil, fmt.Errorf("Failed to update root directory: %s", err)
	}

	return f, nil
}

func createMissingRemoteDirs(drv *drive.Drive, files *syncFiles, args UploadSyncArgs) (*syncFiles, error) {
	missingDirs := files.filterMissingRemoteDirs()
	missingCount := len(missingDirs)

	if missingCount > 0 {
		fmt.Fprintf(args.Out, "\n%d remote directories are missing\n", missingCount)
	}

	// Sort directories so that the dirs with the shortest path comes first
	sort.Sort(byLocalPathLength(missingDirs))

	for i, lf := range missingDirs {
		parentPath := common.ParentFilePath(lf.relPath)
		parent, ok := files.findRemoteByPath(parentPath)
		if !ok {
			return nil, fmt.Errorf("Could not find remote directory with path '%s'", parentPath)
		}

		fmt.Fprintf(args.Out, "[%04d/%04d] Creating directory %s\n", i+1, missingCount, filepath.Join(files.root.file.Name, lf.relPath))

		f, err := createMissingRemoteDir(drv, createMissingRemoteDirArgs{
			name:     lf.info.Name(),
			parentId: parent.file.Id,
			rootId:   args.RootId,
			dryRun:   args.DryRun,
			try:      0,
		})
		if err != nil {
			return nil, err
		}

		files.remote = append(files.remote, &RemoteFile{
			relPath: lf.relPath,
			file:    f,
		})
	}

	return files, nil
}

type createMissingRemoteDirArgs struct {
	name     string
	parentId string
	rootId   string
	dryRun   bool
	try      int
}

func uploadMissingFiles(drv *drive.Drive, missingFiles []*LocalFile, files *syncFiles, args UploadSyncArgs) error {
	missingCount := len(missingFiles)

	if missingCount > 0 {
		fmt.Fprintf(args.Out, "\n%d remote files are missing\n", missingCount)
	}

	for i, lf := range missingFiles {
		parentPath := common.ParentFilePath(lf.relPath)
		parent, ok := files.findRemoteByPath(parentPath)
		if !ok {
			return fmt.Errorf("Could not find remote directory with path '%s'", parentPath)
		}

		fmt.Fprintf(args.Out, "[%04d/%04d] Uploading %s -> %s\n", i+1, missingCount, lf.relPath, filepath.Join(files.root.file.Name, lf.relPath))

		err := uploadMissingFile(drv, parent.file.Id, lf, args, 0)
		if err != nil {
			return err
		}
	}

	return nil
}

func updateChangedFiles(drv *drive.Drive, changedFiles []*changedFile, root *gdrive.File, args UploadSyncArgs) error {
	changedCount := len(changedFiles)

	if changedCount > 0 {
		fmt.Fprintf(args.Out, "\n%d local files has changed\n", changedCount)
	}

	for i, cf := range changedFiles {
		if skip, reason := checkRemoteConflict(cf, args.Resolution); skip {
			fmt.Fprintf(args.Out, "[%04d/%04d] Skipping %s (%s)\n", i+1, changedCount, cf.local.relPath, reason)
			continue
		}

		fmt.Fprintf(args.Out, "[%04d/%04d] Updating %s -> %s\n", i+1, changedCount, cf.local.relPath, filepath.Join(root.Name, cf.local.relPath))

		err := updateChangedFile(drv, cf, args, 0)
		if err != nil {
			return err
		}
	}

	return nil
}

func deleteExtraneousRemoteFiles(drv *drive.Drive, files *syncFiles, args UploadSyncArgs) error {
	extraneousFiles := files.filterExtraneousRemoteFiles()
	extraneousCount := len(extraneousFiles)

	if extraneousCount > 0 {
		fmt.Fprintf(args.Out, "\n%d remote files are extraneous\n", extraneousCount)
	}

	// Sort files so that the files with the longest path comes first
	sort.Sort(sort.Reverse(byRemotePathLength(extraneousFiles)))

	for i, rf := range extraneousFiles {
		fmt.Fprintf(args.Out, "[%04d/%04d] Deleting %s\n", i+1, extraneousCount, filepath.Join(files.root.file.Name, rf.relPath))

		err := deleteRemoteFile(drv, rf, args, 0)
		if err != nil {
			return err
		}
	}

	return nil
}

func createMissingRemoteDir(drv *drive.Drive, args createMissingRemoteDirArgs) (*gdrive.File, error) {
	dstFile := &gdrive.File{
		Name:          args.name,
		MimeType:      common.DirectoryMimeType,
		Parents:       []string{args.parentId},
		AppProperties: map[string]string{"sync": "true", "syncRootId": args.rootId},
	}

	if args.dryRun {
		return dstFile, nil
	}

	f, err := drv.Service.Files.Create(dstFile).Do()
	if err != nil {
		if common.IsBackendOrRateLimitError(err) && args.try < common.MaxErrorRetries {
			common.ExponentialBackoffSleep(args.try)
			args.try++
			return createMissingRemoteDir(drv, args)
		} else {
			return nil, fmt.Errorf("Failed to create directory: %s", err)
		}
	}

	return f, nil
}

func uploadMissingFile(drv *drive.Drive, parentId string, lf *LocalFile, args UploadSyncArgs, try int) error {
	if args.DryRun {
		return nil
	}

	srcFile, err := os.Open(lf.absPath)
	if err != nil {
		return fmt.Errorf("Failed to open file: %s", err)
	}

	// Close file on function exit
	defer srcFile.Close()

	// Instantiate drive file
	dstFile := &gdrive.File{
		Name:          lf.info.Name(),
		Parents:       []string{parentId},
		AppProperties: map[string]string{"sync": "true", "syncRootId": args.RootId},
	}

	// Chunk size option
	chunkSize := googleapi.ChunkSize(int(args.ChunkSize))

	// Wrap file in progress reader
	progressReader := common.GetProgressReader(srcFile, args.Progress, lf.info.Size())

	// Wrap reader in timeout reader
	reader, ctx := common.GetTimeoutReaderContext(progressReader, args.Timeout)

	_, err = drv.Service.Files.Create(dstFile).Fields("id", "name", "size", "md5Checksum").Context(ctx).Media(reader, chunkSize).Do()
	if err != nil {
		if common.IsBackendOrRateLimitError(err) && try < common.MaxErrorRetries {
			common.ExponentialBackoffSleep(try)
			try++
			return uploadMissingFile(drv, parentId, lf, args, try)
		} else if common.IsTimeoutError(err) {
			return fmt.Errorf("Failed to upload file: timeout, no data was transferred for %v", args.Timeout)
		} else {
			return fmt.Errorf("Failed to upload file: %s", err)
		}
	}

	return nil
}

func updateChangedFile(drv *drive.Drive, cf *changedFile, args UploadSyncArgs, try int) error {
	if args.DryRun {
		return nil
	}

	srcFile, err := os.Open(cf.local.absPath)
	if err != nil {
		return fmt.Errorf("Failed to open file: %s", err)
	}

	// Close file on function exit
	defer srcFile.Close()

	// Instantiate drive file
	dstFile := &gdrive.File{}

	// Chunk size option
	chunkSize := googleapi.ChunkSize(int(args.ChunkSize))

	// Wrap file in progress reader
	progressReader := common.GetProgressReader(srcFile, args.Progress, cf.local.info.Size())

	// Wrap reader in timeout reader
	reader, ctx := common.GetTimeoutReaderContext(progressReader, args.Timeout)

	_, err = drv.Service.Files.Update(cf.remote.file.Id, dstFile).Context(ctx).Media(reader, chunkSize).Do()
	if err != nil {
		if common.IsBackendOrRateLimitError(err) && try < common.MaxErrorRetries {
			common.ExponentialBackoffSleep(try)
			try++
			return updateChangedFile(drv, cf, args, try)
		} else if common.IsTimeoutError(err) {
			return fmt.Errorf("Failed to upload file: timeout, no data was transferred for %v", args.Timeout)
		} else {
			return fmt.Errorf("Failed to update file: %s", err)
		}
	}

	return nil
}

func deleteRemoteFile(drv *drive.Drive, rf *RemoteFile, args UploadSyncArgs, try int) error {
	if args.DryRun {
		return nil
	}

	err := drv.Service.Files.Delete(rf.file.Id).Do()
	if err != nil {
		if common.IsBackendOrRateLimitError(err) && try < common.MaxErrorRetries {
			common.ExponentialBackoffSleep(try)
			try++
			return deleteRemoteFile(drv, rf, args, try)
		} else {
			return fmt.Errorf("Failed to delete file: %s", err)
		}
	}

	return nil
}

func dirIsEmpty(drv *drive.Drive, id string) (bool, error) {
	query := fmt.Sprintf("'%s' in parents", id)
	fileList, err := drv.Service.Files.List().Q(query).Do()
	if err != nil {
		return false, fmt.Errorf("Empty dir check failed: %s", err)
	}

	return len(fileList.Files) == 0, nil
}

func checkRemoteConflict(cf *changedFile, resolution ConflictResolution) (bool, string) {
	// No conflict unless remote file was last modified
	if cf.compareModTime() != RemoteLastModified {
		return false, ""
	}

	// Don't skip if want to keep the local file
	if resolution == KeepLocal {
		return false, ""
	}

	// Skip if we want to keep the remote file
	if resolution == KeepRemote {
		return true, "conflicting file, keeping remote file"
	}

	if resolution == KeepLargest {
		largest := cf.compareSize()

		// Skip if the remote file is largest
		if largest == RemoteLargestSize {
			return true, "conflicting file, remote file is largest, keeping remote"
		}

		// Don't skip if the local file is largest
		if largest == LocalLargestSize {
			return false, ""
		}

		// Keep remote if both files have the same size
		if largest == EqualSize {
			return true, "conflicting file, file sizes are equal, keeping remote"
		}
	}

	// The conditionals above should cover all cases,
	// unless the programmer did something wrong,
	// in which case we default to being non-destructive and skip the file
	return true, "conflicting file, unhandled case"
}

func ensureNoRemoteModifications(files []*changedFile) error {
	conflicts := findRemoteConflicts(files)
	if len(conflicts) == 0 {
		return nil
	}

	buffer := bytes.NewBufferString("")
	formatConflicts(conflicts, buffer)
	return fmt.Errorf("%s", buffer.String())
}

func checkRemoteFreeSpace(drv *drive.Drive, missingFiles []*LocalFile, changedFiles []*changedFile) (bool, string) {
	about, err := drv.Service.About.Get().Fields("storageQuota").Do()
	if err != nil {
		return false, fmt.Sprintf("Failed to determine free space: %s", err)
	}

	quota := about.StorageQuota
	if quota.Limit == 0 {
		return true, ""
	}

	freeSpace := quota.Limit - quota.Usage

	var totalSize int64

	for _, lf := range missingFiles {
		totalSize += lf.Size()
	}

	for _, cf := range changedFiles {
		totalSize += cf.local.Size()
	}

	if totalSize > freeSpace {
		return false, fmt.Sprintf("Not enough free space, have %s need %s", common.FormatSize(freeSpace, false), common.FormatSize(totalSize, false))
	}

	return true, ""
}
