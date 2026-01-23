package about

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/imzza/gdrive/internal/drive"
	"github.com/imzza/gdrive/internal/drive/common"
)

type AboutArgs struct {
	Out         io.Writer
	SizeInBytes bool
}

func About(drv *drive.Drive, args AboutArgs) (err error) {
	about, err := drv.Service.About.Get().Fields("maxImportSizes", "maxUploadSize", "storageQuota", "user").Do()
	if err != nil {
		return fmt.Errorf("Failed to get about: %s", err)
	}

	user := about.User
	quota := about.StorageQuota

	fmt.Fprintf(args.Out, "User: %s, %s\n", user.DisplayName, user.EmailAddress)
	fmt.Fprintf(args.Out, "Used: %s\n", common.FormatSize(quota.Usage, args.SizeInBytes))
	fmt.Fprintf(args.Out, "Free: %s\n", common.FormatSize(quota.Limit-quota.Usage, args.SizeInBytes))
	fmt.Fprintf(args.Out, "Total: %s\n", common.FormatSize(quota.Limit, args.SizeInBytes))
	fmt.Fprintf(args.Out, "Max upload size: %s\n", common.FormatSize(about.MaxUploadSize, args.SizeInBytes))
	return
}

func UserEmail(drv *drive.Drive) (string, error) {
	about, err := drv.Service.About.Get().Fields("user").Do()
	if err != nil {
		return "", fmt.Errorf("Failed to get user info: %s", err)
	}

	if about.User == nil || about.User.EmailAddress == "" {
		return "", fmt.Errorf("Failed to get user email")
	}

	return about.User.EmailAddress, nil
}

type AboutImportArgs struct {
	Out io.Writer
}

func AboutImport(drv *drive.Drive, args AboutImportArgs) (err error) {
	about, err := drv.Service.About.Get().Fields("importFormats").Do()
	if err != nil {
		return fmt.Errorf("Failed to get about: %s", err)
	}
	printAboutFormats(args.Out, about.ImportFormats)
	return
}

type AboutExportArgs struct {
	Out io.Writer
}

func AboutExport(drv *drive.Drive, args AboutExportArgs) (err error) {
	about, err := drv.Service.About.Get().Fields("exportFormats").Do()
	if err != nil {
		return fmt.Errorf("Failed to get about: %s", err)
	}
	printAboutFormats(args.Out, about.ExportFormats)
	return
}

func printAboutFormats(out io.Writer, formats map[string][]string) {
	w := new(tabwriter.Writer)
	w.Init(out, 0, 0, 3, ' ', 0)

	fmt.Fprintln(w, "From\tTo")

	for from, toFormats := range formats {
		fmt.Fprintf(w, "%s\t%s\n", from, common.FormatList(toFormats))
	}

	w.Flush()
}
