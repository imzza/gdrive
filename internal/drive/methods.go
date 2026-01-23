package drive

import (
	"github.com/imzza/gdrive/internal/drive/about"
	"github.com/imzza/gdrive/internal/drive/changes"
	"github.com/imzza/gdrive/internal/drive/drives"
	"github.com/imzza/gdrive/internal/drive/files"
	"github.com/imzza/gdrive/internal/drive/permissions"
	"github.com/imzza/gdrive/internal/drive/revision"
	syncdrive "github.com/imzza/gdrive/internal/drive/sync"
)

func (d *Drive) About(args about.AboutArgs) error {
	return about.About(d, args)
}

func (d *Drive) UserEmail() (string, error) {
	return about.UserEmail(d)
}

func (d *Drive) ListFiles(args files.ListArgs) error {
	return files.List(d, args)
}

func (d *Drive) Info(args files.InfoArgs) error {
	return files.Info(d, args)
}

func (d *Drive) Mkdir(args files.MkdirArgs) error {
	return files.Mkdir(d, args)
}

func (d *Drive) Rename(args files.RenameArgs) error {
	return files.Rename(d, args)
}

func (d *Drive) Move(args files.MoveArgs) error {
	return files.Move(d, args)
}

func (d *Drive) Copy(args files.CopyArgs) error {
	return files.Copy(d, args)
}

func (d *Drive) Delete(args files.DeleteArgs) error {
	return files.Delete(d, args)
}

func (d *Drive) Download(args files.DownloadArgs) error {
	return files.Download(d, args)
}

func (d *Drive) DownloadQuery(args files.DownloadQueryArgs) error {
	return files.DownloadQuery(d, args)
}

func (d *Drive) Upload(args files.UploadArgs) error {
	return files.Upload(d, args)
}

func (d *Drive) UploadStream(args files.UploadStreamArgs) error {
	return files.UploadStream(d, args)
}

func (d *Drive) Update(args files.UpdateArgs) error {
	return files.Update(d, args)
}

func (d *Drive) Import(args files.ImportArgs) error {
	return files.Import(d, args)
}

func (d *Drive) Export(args files.ExportArgs) error {
	return files.Export(d, args)
}

func (d *Drive) Share(args permissions.ShareArgs) error {
	return permissions.Share(d, args)
}

func (d *Drive) ListPermissions(args permissions.ListPermissionsArgs) error {
	return permissions.ListPermissions(d, args)
}

func (d *Drive) RevokePermission(args permissions.RevokePermissionArgs) error {
	return permissions.RevokePermission(d, args)
}

func (d *Drive) ListChanges(args changes.ListChangesArgs) error {
	return changes.ListChanges(d, args)
}

func (d *Drive) ListRevisions(args revision.ListRevisionsArgs) error {
	return revision.ListRevisions(d, args)
}

func (d *Drive) DownloadRevision(args revision.DownloadRevisionArgs) error {
	return revision.DownloadRevision(d, args)
}

func (d *Drive) DeleteRevision(args revision.DeleteRevisionArgs) error {
	return revision.DeleteRevision(d, args)
}

func (d *Drive) ListSync(args syncdrive.ListSyncArgs) error {
	return syncdrive.ListSync(d, args)
}

func (d *Drive) ListRecursiveSync(args syncdrive.ListRecursiveSyncArgs) error {
	return syncdrive.ListRecursiveSync(d, args)
}

func (d *Drive) DownloadSync(args syncdrive.DownloadSyncArgs) error {
	return syncdrive.DownloadSync(d, args)
}

func (d *Drive) UploadSync(args syncdrive.UploadSyncArgs) error {
	return syncdrive.UploadSync(d, args)
}

func (d *Drive) ListDrives(args drives.ListDrivesArgs) error {
	return drives.ListDrives(d, args)
}
