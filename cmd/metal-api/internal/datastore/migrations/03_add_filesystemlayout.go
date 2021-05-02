package migrations

import (
	"fmt"

	r "gopkg.in/rethinkdb/rethinkdb-go.v6"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
)

func init() {
	datastore.MustRegisterMigration(datastore.Migration{
		Name:    "add a filesystemlayout to already allocated machines",
		Version: 3,
		Up: func(db *r.Term, session r.QueryExecutor, rs *datastore.RethinkStore) error {
			gptboot := metal.GPTBoot
			gptlinux := metal.GPTLinux
			gptraid := metal.GPTLinuxRaid
			tmpfs := metal.Filesystem{Path: strPtr("/tmp"), Device: "tmpfs", Format: metal.TMPFS, MountOptions: []string{"defaults", "noatime", "nosuid", "nodev", "noexec", "mode=1777", "size=512M"}}
			fsls := metal.FilesystemLayouts{}
			legacyDefault := &metal.FilesystemLayout{
				Base: metal.Base{ID: "legacy-default", Name: "legacy filesystemlayout"},
				Disks: []metal.Disk{
					{
						Device: "/dev/sda",
						Partitions: []metal.DiskPartition{
							{Number: 1, Label: strPtr("efi"), Size: 500, GPTType: &gptboot},
							{Number: 2, Label: strPtr("root"), Size: 5000, GPTType: &gptlinux},
							{Number: 3, Label: strPtr("varlib"), Size: 0, GPTType: &gptlinux},
						},
					},
				},
				Filesystems: []metal.Filesystem{
					{Path: strPtr("/boot/efi"), Device: "/dev/sda1", Format: metal.VFAT, Label: strPtr("efi"), CreateOptions: []string{"-F", "32"}},
					{Path: strPtr("/"), Device: "/dev/sda2", Format: metal.EXT4, Label: strPtr("root")},
					{Path: strPtr("/var/lib"), Device: "/dev/sda2", Format: metal.EXT4, Label: strPtr("varlib")},
					tmpfs,
				},
			}
			legacyS2 := &metal.FilesystemLayout{
				Base: metal.Base{ID: "legacy-s2", Name: "legacy filesystemlayout for s2 machines"},
				Disks: []metal.Disk{
					{
						Device: "/dev/sde",
						Partitions: []metal.DiskPartition{
							{Number: 1, Label: strPtr("efi"), Size: 500, GPTType: &gptboot},
							{Number: 2, Label: strPtr("root"), Size: 5000, GPTType: &gptlinux},
							{Number: 3, Label: strPtr("varlib"), Size: 0, GPTType: &gptlinux},
						},
					},
				},
				Filesystems: []metal.Filesystem{
					{Path: strPtr("/boot/efi"), Device: "/dev/sde1", Format: metal.VFAT, Label: strPtr("efi"), CreateOptions: []string{"-F", "32"}},
					{Path: strPtr("/"), Device: "/dev/sde2", Format: metal.EXT4, Label: strPtr("root")},
					{Path: strPtr("/var/lib"), Device: "/dev/sde2", Format: metal.EXT4, Label: strPtr("varlib")},
					tmpfs,
				},
			}
			legacyS3 := &metal.FilesystemLayout{
				Base: metal.Base{ID: "legacy-s3", Name: "legacy filesystemlayout for s3 machines"},
				Disks: []metal.Disk{
					{
						Device: "/dev/sda",
						Partitions: []metal.DiskPartition{
							{Number: 1, Label: strPtr("efi"), Size: 500, GPTType: &gptraid},
							{Number: 2, Label: strPtr("root"), Size: 50000, GPTType: &gptraid},
							{Number: 3, Label: strPtr("var"), Size: 0, GPTType: &gptraid},
						},
					},
					{
						Device: "/dev/sdb",
						Partitions: []metal.DiskPartition{
							{Number: 1, Label: strPtr("efi"), Size: 500, GPTType: &gptraid},
							{Number: 2, Label: strPtr("root"), Size: 50000, GPTType: &gptraid},
							{Number: 3, Label: strPtr("var"), Size: 0, GPTType: &gptraid},
						},
					},
				},
				Raid: []metal.Raid{
					{ArrayName: "/dev/md0", Devices: []string{"/dev/sda1", "/dev/sdb1"}},
					{ArrayName: "/dev/md1", Devices: []string{"/dev/sda2", "/dev/sdb2"}},
					{ArrayName: "/dev/md2", Devices: []string{"/dev/sda3", "/dev/sdb3"}},
				},
				Filesystems: []metal.Filesystem{
					{Path: strPtr("/boot/efi"), Device: "/dev/md1", Format: metal.VFAT, Label: strPtr("efi"), CreateOptions: []string{"-F", "32"}},
					{Path: strPtr("/"), Device: "/dev/md2", Format: metal.EXT4, Label: strPtr("root")},
					{Path: strPtr("/var"), Device: "/dev/md3", Format: metal.EXT4, Label: strPtr("varlib")},
					tmpfs,
				},
			}
			fsls = append(fsls, *legacyDefault)
			fsls = append(fsls, *legacyS2)
			fsls = append(fsls, *legacyS3)

			for i := range fsls {
				fsl := fsls[i]
				rs.SugaredLogger.Infow("create filesystemlayout", "id", fsl.ID)
				err := rs.CreateFilesystemLayout(&fsl)
				if err != nil {
					return err
				}
			}

			machines, err := rs.ListMachines()
			if err != nil {
				return err
			}

			for i := range machines {
				old := machines[i]
				if old.Allocation == nil {
					continue
				}
				if old.Allocation.FilesystemLayout != nil {
					continue
				}

				var fsl *metal.FilesystemLayout
				var err error
				switch old.SizeID {
				case "s2-xlarge-x86":
					fsl, err = rs.FindFilesystemLayout("legacy-s2")
					if err != nil {
						return fmt.Errorf("unable to select filesystemlayout for machine:%s size:%s error,%w", old.ID, old.SizeID, err)
					}
				case "s3-xlarge-x86":
					fsl, err = rs.FindFilesystemLayout("legacy-s3")
					if err != nil {
						return fmt.Errorf("unable to select filesystemlayout for machine:%s size:%s error,%w", old.ID, old.SizeID, err)
					}
				default:
					fsl, err = rs.FindFilesystemLayout("legacy-default")
					if err != nil {
						return fmt.Errorf("unable to select filesystemlayout for machine:%s size:%s error,%w", old.ID, old.SizeID, err)
					}
				}

				n := old
				n.Allocation.FilesystemLayout = fsl
				rs.SugaredLogger.Infow("set filesystemlayout to machine allocation", "machineID", n.ID, "layout", fsl.ID)
				err = rs.UpdateMachine(&old, &n)
				if err != nil {
					return err
				}
			}
			return nil
		},
	})
}

func strPtr(s string) *string {
	return &s
}
