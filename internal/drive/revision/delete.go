package revision

import (
	"fmt"
	"io"

	"github.com/imzza/gdrive/internal/drive"
)

type DeleteRevisionArgs struct {
	Out        io.Writer
	FileId     string
	RevisionId string
}

func DeleteRevision(drv *drive.Drive, args DeleteRevisionArgs) (err error) {
	rev, err := drv.Service.Revisions.Get(args.FileId, args.RevisionId).Fields("originalFilename").Do()
	if err != nil {
		return fmt.Errorf("Failed to get revision: %s", err)
	}

	if rev.OriginalFilename == "" {
		return fmt.Errorf("Deleting revisions for this file type is not supported")
	}

	err = drv.Service.Revisions.Delete(args.FileId, args.RevisionId).Do()
	if err != nil {
		return fmt.Errorf("Failed to delete revision: %s", err)
	}

	fmt.Fprintf(args.Out, "Deleted revision '%s'\n", args.RevisionId)
	return
}
