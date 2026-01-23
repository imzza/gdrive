package permissions

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/imzza/gdrive/internal/drive"
	"github.com/imzza/gdrive/internal/drive/common"
	gdrive "google.golang.org/api/drive/v3"
)

type ShareArgs struct {
	Out          io.Writer
	FileId       string
	Role         string
	Type         string
	Email        string
	Domain       string
	Discoverable bool
}

func Share(drv *drive.Drive, args ShareArgs) error {
	permission := &gdrive.Permission{
		AllowFileDiscovery: args.Discoverable,
		Role:               args.Role,
		Type:               args.Type,
		EmailAddress:       args.Email,
		Domain:             args.Domain,
	}

	_, err := drv.Service.Permissions.Create(args.FileId, permission).Do()
	if err != nil {
		return fmt.Errorf("Failed to share file: %s", err)
	}

	fmt.Fprintf(args.Out, "Granted %s permission to %s\n", args.Role, args.Type)
	return nil
}

type RevokePermissionArgs struct {
	Out          io.Writer
	FileId       string
	PermissionId string
}

func RevokePermission(drv *drive.Drive, args RevokePermissionArgs) error {
	err := drv.Service.Permissions.Delete(args.FileId, args.PermissionId).Do()
	if err != nil {
		return fmt.Errorf("Failed to revoke permission: %s", err)
	}

	fmt.Fprintf(args.Out, "Permission revoked\n")
	return nil
}

type ListPermissionsArgs struct {
	Out    io.Writer
	FileId string
}

func ListPermissions(drv *drive.Drive, args ListPermissionsArgs) error {
	permList, err := drv.Service.Permissions.List(args.FileId).Fields("permissions(id,role,type,domain,emailAddress,allowFileDiscovery)").Do()
	if err != nil {
		return fmt.Errorf("Failed to list permissions: %s", err)
	}

	printPermissions(printPermissionsArgs{
		out:         args.Out,
		permissions: permList.Permissions,
	})
	return nil
}

func ShareAnyoneReader(drv *drive.Drive, fileId string) error {
	permission := &gdrive.Permission{
		Role: "reader",
		Type: "anyone",
	}

	_, err := drv.Service.Permissions.Create(fileId, permission).Do()
	if err != nil {
		return fmt.Errorf("Failed to share file: %s", err)
	}

	return nil
}

type printPermissionsArgs struct {
	out         io.Writer
	permissions []*gdrive.Permission
}

func printPermissions(args printPermissionsArgs) {
	w := new(tabwriter.Writer)
	w.Init(args.out, 0, 0, 3, ' ', 0)

	fmt.Fprintln(w, "Id\tType\tRole\tEmail\tDomain\tDiscoverable")

	for _, p := range args.permissions {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			p.Id,
			p.Type,
			p.Role,
			p.EmailAddress,
			p.Domain,
			common.FormatBool(p.AllowFileDiscovery),
		)
	}

	w.Flush()
}
