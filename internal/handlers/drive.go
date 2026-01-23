package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/imzza/gdrive/internal/auth"
	"github.com/imzza/gdrive/internal/cli"
	drivepkg "github.com/imzza/gdrive/internal/drive"
	"github.com/imzza/gdrive/internal/drive/about"
	"github.com/imzza/gdrive/internal/drive/changes"
	"github.com/imzza/gdrive/internal/drive/files"
	"github.com/imzza/gdrive/internal/drive/permissions"
	"github.com/imzza/gdrive/internal/drive/revision"
	syncdrive "github.com/imzza/gdrive/internal/drive/sync"
	"github.com/imzza/gdrive/internal/utils"
)

const ClientId = "367116221053-7n0vf5akeru7on6o2fjinrecpdoe99eg.apps.googleusercontent.com"
const ClientSecret = "1qsNodXNaWq1mQuBjUjmvhoO"
const TokenFilename = "tokens.json"
const DefaultCacheFileName = "file_cache.json"

func ListHandler(ctx cli.Context) {
	args := ctx.Args()
	drv := newDrive(args)
	err := drv.ListFiles(files.ListArgs{
		Out:         os.Stdout,
		MaxFiles:    args.Int64("maxFiles"),
		NameWidth:   args.Int64("nameWidth"),
		Query:       args.String("query"),
		SortOrder:   args.String("sortOrder"),
		SkipHeader:  args.Bool("skipHeader"),
		SizeInBytes: args.Bool("sizeInBytes"),
		AbsPath:     args.Bool("absPath"),
	})
	utils.CheckErr(err)
}

func RenameHandler(ctx cli.Context) {
	args := ctx.Args()
	drv := newDrive(args)
	err := drv.Rename(files.RenameArgs{
		Out:  os.Stdout,
		Id:   args.String("fileId"),
		Name: args.String("name"),
	})
	utils.CheckErr(err)
}

func MoveHandler(ctx cli.Context) {
	args := ctx.Args()
	drv := newDrive(args)
	err := drv.Move(files.MoveArgs{
		Out:      os.Stdout,
		Id:       args.String("fileId"),
		FolderId: args.String("folderId"),
	})
	utils.CheckErr(err)
}

func CopyHandler(ctx cli.Context) {
	args := ctx.Args()
	drv := newDrive(args)
	err := drv.Copy(files.CopyArgs{
		Out:      os.Stdout,
		Id:       args.String("fileId"),
		FolderId: args.String("folderId"),
	})
	utils.CheckErr(err)
}

func ListChangesHandler(ctx cli.Context) {
	args := ctx.Args()
	drv := newDrive(args)
	err := drv.ListChanges(changes.ListChangesArgs{
		Out:        os.Stdout,
		PageToken:  args.String("pageToken"),
		MaxChanges: args.Int64("maxChanges"),
		Now:        args.Bool("now"),
		NameWidth:  args.Int64("nameWidth"),
		SkipHeader: args.Bool("skipHeader"),
	})
	utils.CheckErr(err)
}

func DownloadHandler(ctx cli.Context) {
	args := ctx.Args()
	checkDownloadArgs(args)
	drv := newDrive(args)
	err := drv.Download(files.DownloadArgs{
		Out:       os.Stdout,
		Id:        args.String("fileId"),
		Force:     args.Bool("force"),
		Skip:      args.Bool("skip"),
		Path:      args.String("path"),
		Delete:    args.Bool("delete"),
		Recursive: args.Bool("recursive"),
		Stdout:    args.Bool("stdout"),
		Progress:  progressWriter(args.Bool("noProgress")),
		Timeout:   durationInSeconds(args.Int64("timeout")),
	})
	utils.CheckErr(err)
}

func DownloadQueryHandler(ctx cli.Context) {
	args := ctx.Args()
	drv := newDrive(args)
	err := drv.DownloadQuery(files.DownloadQueryArgs{
		Out:       os.Stdout,
		Query:     args.String("query"),
		Force:     args.Bool("force"),
		Skip:      args.Bool("skip"),
		Recursive: args.Bool("recursive"),
		Path:      args.String("path"),
		Progress:  progressWriter(args.Bool("noProgress")),
	})
	utils.CheckErr(err)
}

func DownloadSyncHandler(ctx cli.Context) {
	args := ctx.Args()
	configDir := getConfigDir(args)
	cachePath := filepath.Join(configDir, DefaultCacheFileName)
	drv := newDrive(args)
	err := drv.DownloadSync(syncdrive.DownloadSyncArgs{
		Out:              os.Stdout,
		Progress:         progressWriter(args.Bool("noProgress")),
		Path:             args.String("path"),
		RootId:           args.String("fileId"),
		DryRun:           args.Bool("dryRun"),
		DeleteExtraneous: args.Bool("deleteExtraneous"),
		Timeout:          durationInSeconds(args.Int64("timeout")),
		Resolution:       conflictResolution(args),
		Comparer:         NewCachedMd5Comparer(cachePath),
	})
	utils.CheckErr(err)
}

func DownloadRevisionHandler(ctx cli.Context) {
	args := ctx.Args()
	drv := newDrive(args)
	err := drv.DownloadRevision(revision.DownloadRevisionArgs{
		Out:        os.Stdout,
		FileId:     args.String("fileId"),
		RevisionId: args.String("revId"),
		Force:      args.Bool("force"),
		Stdout:     args.Bool("stdout"),
		Path:       args.String("path"),
		Progress:   progressWriter(args.Bool("noProgress")),
		Timeout:    durationInSeconds(args.Int64("timeout")),
	})
	utils.CheckErr(err)
}

func UploadHandler(ctx cli.Context) {
	args := ctx.Args()
	checkUploadArgs(args)
	drv := newDrive(args)
	err := drv.Upload(files.UploadArgs{
		Out:         os.Stdout,
		Progress:    progressWriter(args.Bool("noProgress")),
		Path:        args.String("path"),
		Name:        args.String("name"),
		Description: args.String("description"),
		Parents:     args.StringSlice("parent"),
		Mime:        args.String("mime"),
		Recursive:   args.Bool("recursive"),
		Share:       args.Bool("share"),
		Delete:      args.Bool("delete"),
		ChunkSize:   args.Int64("chunksize"),
		Timeout:     durationInSeconds(args.Int64("timeout")),
	})
	utils.CheckErr(err)
}

func UploadStdinHandler(ctx cli.Context) {
	args := ctx.Args()
	drv := newDrive(args)
	err := drv.UploadStream(files.UploadStreamArgs{
		Out:         os.Stdout,
		In:          os.Stdin,
		Name:        args.String("name"),
		Description: args.String("description"),
		Parents:     args.StringSlice("parent"),
		Mime:        args.String("mime"),
		Share:       args.Bool("share"),
		ChunkSize:   args.Int64("chunksize"),
		Timeout:     durationInSeconds(args.Int64("timeout")),
		Progress:    progressWriter(args.Bool("noProgress")),
	})
	utils.CheckErr(err)
}

func UploadSyncHandler(ctx cli.Context) {
	args := ctx.Args()
	configDir := getConfigDir(args)
	cachePath := filepath.Join(configDir, DefaultCacheFileName)
	drv := newDrive(args)
	err := drv.UploadSync(syncdrive.UploadSyncArgs{
		Out:              os.Stdout,
		Progress:         progressWriter(args.Bool("noProgress")),
		Path:             args.String("path"),
		RootId:           args.String("fileId"),
		DryRun:           args.Bool("dryRun"),
		DeleteExtraneous: args.Bool("deleteExtraneous"),
		ChunkSize:        args.Int64("chunksize"),
		Timeout:          durationInSeconds(args.Int64("timeout")),
		Resolution:       conflictResolution(args),
		Comparer:         NewCachedMd5Comparer(cachePath),
	})
	utils.CheckErr(err)
}

func UpdateHandler(ctx cli.Context) {
	args := ctx.Args()
	drv := newDrive(args)
	err := drv.Update(files.UpdateArgs{
		Out:         os.Stdout,
		Id:          args.String("fileId"),
		Path:        args.String("path"),
		Name:        args.String("name"),
		Description: args.String("description"),
		Parents:     args.StringSlice("parent"),
		Mime:        args.String("mime"),
		Progress:    progressWriter(args.Bool("noProgress")),
		ChunkSize:   args.Int64("chunksize"),
		Timeout:     durationInSeconds(args.Int64("timeout")),
	})
	utils.CheckErr(err)
}

func InfoHandler(ctx cli.Context) {
	args := ctx.Args()
	drv := newDrive(args)
	err := drv.Info(files.InfoArgs{
		Out:         os.Stdout,
		Id:          args.String("fileId"),
		SizeInBytes: args.Bool("sizeInBytes"),
	})
	utils.CheckErr(err)
}

func ImportHandler(ctx cli.Context) {
	args := ctx.Args()
	drv := newDrive(args)
	err := drv.Import(files.ImportArgs{
		Mime:     args.String("mime"),
		Out:      os.Stdout,
		Path:     args.String("path"),
		Parents:  args.StringSlice("parent"),
		Progress: progressWriter(args.Bool("noProgress")),
	})
	utils.CheckErr(err)
}

func ExportHandler(ctx cli.Context) {
	args := ctx.Args()
	drv := newDrive(args)
	err := drv.Export(files.ExportArgs{
		Out:        os.Stdout,
		Id:         args.String("fileId"),
		Mime:       args.String("mime"),
		PrintMimes: args.Bool("printMimes"),
		Force:      args.Bool("force"),
	})
	utils.CheckErr(err)
}

func ListRevisionsHandler(ctx cli.Context) {
	args := ctx.Args()
	drv := newDrive(args)
	err := drv.ListRevisions(revision.ListRevisionsArgs{
		Out:         os.Stdout,
		Id:          args.String("fileId"),
		NameWidth:   args.Int64("nameWidth"),
		SizeInBytes: args.Bool("sizeInBytes"),
		SkipHeader:  args.Bool("skipHeader"),
	})
	utils.CheckErr(err)
}

func MkdirHandler(ctx cli.Context) {
	args := ctx.Args()
	drv := newDrive(args)
	err := drv.Mkdir(files.MkdirArgs{
		Out:         os.Stdout,
		Name:        args.String("name"),
		Description: args.String("description"),
		Parents:     args.StringSlice("parent"),
	})
	utils.CheckErr(err)
}

func ShareHandler(ctx cli.Context) {
	args := ctx.Args()
	drv := newDrive(args)
	err := drv.Share(permissions.ShareArgs{
		Out:          os.Stdout,
		FileId:       args.String("fileId"),
		Role:         args.String("role"),
		Type:         args.String("type"),
		Email:        args.String("email"),
		Domain:       args.String("domain"),
		Discoverable: args.Bool("discoverable"),
	})
	utils.CheckErr(err)
}

func ShareListHandler(ctx cli.Context) {
	args := ctx.Args()
	drv := newDrive(args)
	err := drv.ListPermissions(permissions.ListPermissionsArgs{
		Out:    os.Stdout,
		FileId: args.String("fileId"),
	})
	utils.CheckErr(err)
}

func ShareRevokeHandler(ctx cli.Context) {
	args := ctx.Args()
	drv := newDrive(args)
	err := drv.RevokePermission(permissions.RevokePermissionArgs{
		Out:          os.Stdout,
		FileId:       args.String("fileId"),
		PermissionId: args.String("permissionId"),
	})
	utils.CheckErr(err)
}

func DeleteHandler(ctx cli.Context) {
	args := ctx.Args()
	drv := newDrive(args)
	err := drv.Delete(files.DeleteArgs{
		Out:       os.Stdout,
		Id:        args.String("fileId"),
		Recursive: args.Bool("recursive"),
	})
	utils.CheckErr(err)
}

func ListSyncHandler(ctx cli.Context) {
	args := ctx.Args()
	drv := newDrive(args)
	err := drv.ListSync(syncdrive.ListSyncArgs{
		Out:        os.Stdout,
		SkipHeader: args.Bool("skipHeader"),
	})
	utils.CheckErr(err)
}

func ListRecursiveSyncHandler(ctx cli.Context) {
	args := ctx.Args()
	drv := newDrive(args)
	err := drv.ListRecursiveSync(syncdrive.ListRecursiveSyncArgs{
		Out:         os.Stdout,
		RootId:      args.String("fileId"),
		SkipHeader:  args.Bool("skipHeader"),
		PathWidth:   args.Int64("pathWidth"),
		SizeInBytes: args.Bool("sizeInBytes"),
		SortOrder:   args.String("sortOrder"),
	})
	utils.CheckErr(err)
}

func DeleteRevisionHandler(ctx cli.Context) {
	args := ctx.Args()
	drv := newDrive(args)
	err := drv.DeleteRevision(revision.DeleteRevisionArgs{
		Out:        os.Stdout,
		FileId:     args.String("fileId"),
		RevisionId: args.String("revId"),
	})
	utils.CheckErr(err)
}

func AboutHandler(ctx cli.Context) {
	args := ctx.Args()
	printAboutHeader()
	fmt.Println("")

	if !hasAuthArgs(args) {
		baseDir := getBaseConfigDir(args)
		accounts, err := listAccounts(baseDir)
		if err != nil {
			utils.ExitF("Failed to list accounts: %s", err)
		}
		if len(accounts) == 0 {
			fmt.Println("No accounts found. Use `gdrive account add` to add an account.")
			return
		}
		config, err := loadAccountConfig(baseDir)
		if err != nil || config.Current == "" {
			fmt.Println("No account selected. Use `gdrive account switch` to select an account.")
			return
		}
	}

	fmt.Println("")
	drv := newDrive(args)
	err := drv.About(about.AboutArgs{
		Out:         os.Stdout,
		SizeInBytes: args.Bool("sizeInBytes"),
	})
	utils.CheckErr(err)
}

func getOauthClient(args cli.Arguments) (*http.Client, error) {
	configDir := getConfigDir(args)
	return getOauthClientWithConfigDir(args, configDir)
}

func getOauthClientWithConfigDir(args cli.Arguments, configDir string) (*http.Client, error) {
	if args.String("refreshToken") != "" && args.String("accessToken") != "" {
		utils.ExitF("Access token not needed when refresh token is provided")
	}

	clientId := ClientId
	clientSecret := ClientSecret
	if secret, err := utils.LoadAccountSecret(configDir); err == nil {
		if secret.ClientID != "" && secret.ClientSecret != "" {
			clientId = secret.ClientID
			clientSecret = secret.ClientSecret
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("Failed to read secret.json: %s", err)
	}

	if args.String("refreshToken") != "" {
		return auth.NewRefreshTokenClient(clientId, clientSecret, args.String("refreshToken")), nil
	}

	if args.String("accessToken") != "" {
		return auth.NewAccessTokenClient(clientId, clientSecret, args.String("accessToken")), nil
	}

	if args.String("serviceAccount") != "" {
		serviceAccountPath := utils.ConfigFilePath(configDir, args.String("serviceAccount"))
		serviceAccountClient, err := auth.NewServiceAccountClient(serviceAccountPath)
		if err != nil {
			return nil, err
		}
		return serviceAccountClient, nil
	}

	tokenPath := utils.ConfigFilePath(configDir, TokenFilename)
	return auth.NewFileSourceClient(clientId, clientSecret, tokenPath, authCodePrompt)
}

func getConfigDir(args cli.Arguments) string {
	baseDir := getBaseConfigDir(args)
	configDir, err := resolveActiveConfigDir(baseDir)
	if err != nil {
		utils.ExitF("%s", err)
	}
	return configDir
}

func newDrive(args cli.Arguments) *drivepkg.Drive {
	oauth, err := getOauthClient(args)
	if err != nil {
		utils.ExitF("Failed getting oauth client: %s", err.Error())
	}

	client, err := drivepkg.New(oauth)
	if err != nil {
		utils.ExitF("Failed getting drive: %s", err.Error())
	}

	return client
}

func authCodePrompt(url string) func() string {
	return func() string {
		fmt.Println("")
		fmt.Println("Gdrive requires permissions to manage your files on Google Drive.")
		fmt.Println("Open the url in your browser and follow the instructions:")
		fmt.Println(url)
		fmt.Println("")
		fmt.Print("Enter verification code: ")

		var code string
		if _, err := fmt.Scan(&code); err != nil {
			fmt.Printf("Failed reading code: %s", err.Error())
		}
		return code
	}
}

func printAboutHeader() {
	fmt.Println("gdrive is a command line application for interacting with Google Drive.")
	fmt.Println("")
	fmt.Println("For the latest information check out the project page: https://github.com/glotlabs/gdrive")
	fmt.Println("You will also find link to the community chat and information on how to support the project.")
}

func hasAuthArgs(args cli.Arguments) bool {
	return args.String("refreshToken") != "" ||
		args.String("accessToken") != "" ||
		args.String("serviceAccount") != ""
}

func progressWriter(discard bool) io.Writer {
	if discard {
		return io.Discard
	}
	return os.Stderr
}

func durationInSeconds(seconds int64) time.Duration {
	return time.Second * time.Duration(seconds)
}

func conflictResolution(args cli.Arguments) syncdrive.ConflictResolution {
	keepLocal := args.Bool("keepLocal")
	keepRemote := args.Bool("keepRemote")
	keepLargest := args.Bool("keepLargest")

	if (keepLocal && keepRemote) || (keepLocal && keepLargest) || (keepRemote && keepLargest) {
		utils.ExitF("Only one conflict resolution flag can be given")
	}

	if keepLocal {
		return syncdrive.KeepLocal
	}

	if keepRemote {
		return syncdrive.KeepRemote
	}

	if keepLargest {
		return syncdrive.KeepLargest
	}

	return syncdrive.NoResolution
}

func checkUploadArgs(args cli.Arguments) {
	if args.Bool("recursive") && args.Bool("delete") {
		utils.ExitF("--delete is not allowed for recursive uploads")
	}

	if args.Bool("recursive") && args.Bool("share") {
		utils.ExitF("--share is not allowed for recursive uploads")
	}
}

func checkDownloadArgs(args cli.Arguments) {
	if args.Bool("recursive") && args.Bool("delete") {
		utils.ExitF("--delete is not allowed for recursive downloads")
	}
}
