package handlers

import (
	"os"

	"github.com/imzza/gdrive/internal/cli"
	"github.com/imzza/gdrive/internal/drive/drives"
	"github.com/imzza/gdrive/internal/utils"
)

func DrivesListHandler(ctx cli.Context) {
	args := ctx.Args()
	drv := newDrive(args)
	err := drv.ListDrives(drives.ListDrivesArgs{
		Out:            os.Stdout,
		SkipHeader:     args.Bool("skipHeader"),
		FieldSeparator: args.String("fieldSeparator"),
	})
	utils.CheckErr(err)
}
