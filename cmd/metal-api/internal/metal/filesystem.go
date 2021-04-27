package metal

import (
	"fmt"
	"path"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/utils"
)

const (
	// VFAT is used for the UEFI boot partition
	VFAT = Format("vfat")
	// EXT3 is usually only used for /boot
	EXT3 = Format("ext3")
	// EXT4 is the default fs
	EXT4 = Format("ext4")
	// SWAP is for the swap partition
	SWAP = Format("swap")
	// TMPFS is used for a memory filesystem typically /tmp
	TMPFS = Format("tmpfs")
	// None
	NONE = Format("none")

	// GPTBoot EFI Boot Partition
	GPTBoot = GPTType("ef00")
	// GPTLinux Linux Partition
	GPTLinux = GPTType("8300")
	// GPTLinuxRaid Linux Raid Partition
	GPTLinuxRaid = GPTType("fd00")
	// GPTLinux Linux Partition
	GPTLinuxLVM = GPTType("8e00")

	// RaidLevel0 is a stripe of two or more disks
	RaidLevel0 = RaidLevel("0")
	// RaidLevel1 is a mirror of two disks
	RaidLevel1 = RaidLevel("1")

	// LVMTypeLinear stripe across all physical volumes
	LVMTypeLinear = LVMType("linear")
	// LVMTypeStripe mirror with raid across all physical volumes
	LVMTypeRaid1 = LVMType("raid1")
)

var (
	SupportedFormats    = map[Format]bool{VFAT: true, EXT3: true, EXT4: true, SWAP: true, TMPFS: true, NONE: true}
	SupportedGPTTypes   = map[GPTType]bool{GPTBoot: true, GPTLinux: true, GPTLinuxLVM: true, GPTLinuxRaid: true}
	SupportedRaidLevels = map[RaidLevel]bool{RaidLevel0: true, RaidLevel1: true}
	SupportedLVMTypes   = map[LVMType]bool{LVMTypeLinear: true, LVMTypeRaid1: true}
)

type (
	// FilesystemLayouts is a slice of FilesystemLayout
	FilesystemLayouts []FilesystemLayout
	// FilesystemLayout to be created on the given machine
	FilesystemLayout struct {
		Base
		// Filesystems to create on the server
		Filesystems []Filesystem
		// Disks to configure in the server with their partitions
		Disks []Disk
		// Raid if not empty, create raid arrays out of the individual disks, to place filesystems onto
		Raid []Raid
		// VolumeGroups to create
		VolumeGroups []VolumeGroup
		// LogicalVolumes to create on top of VolumeGroups
		LogicalVolumes []LogicalVolume
		// Constraints which must match to select this Layout
		Constraints FilesystemLayoutConstraints
	}

	FilesystemLayoutConstraints struct {
		// Sizes defines the list of sizes this layout applies to
		Sizes []string
		// Images defines a map from os to versionconstraint
		// the combination of os and versionconstraint per size must be conflict free over all filesystemlayouts
		Images map[string]string
	}

	RaidLevel string
	Format    string
	GPTType   string
	LVMType   string

	// Filesystem defines a single filesystem to be mounted
	Filesystem struct {
		// Path defines the mountpoint, if nil, it will not be mounted
		Path *string
		// Device where the filesystem is created on, must be the full device path seen by the OS
		Device string
		// Format is the type of filesystem should be created
		Format Format
		// Label is optional enhances readability
		Label *string
		// MountOptions which might be required
		MountOptions []string
		// CreateOptions during filesystem creation
		CreateOptions []string
	}

	// Disk represents a single block device visible from the OS, required
	Disk struct {
		// Device is the full device path
		Device string
		// PartitionPrefix specifies which prefix is used if device is partitioned
		// e.g. device /dev/sda, first partition will be /dev/sda1, prefix is therefore /dev/sda
		// for nvme drives this is different, the prefix there is typically /dev/nvme0n1p
		PartitionPrefix string
		// Partitions to create on this device
		Partitions []DiskPartition2
		// WipeOnReinstall, if set to true the whole disk will be erased if reinstall happens
		// during fresh install all disks are wiped
		WipeOnReinstall bool
	}

	// Raid is optional, if given the devices must match.
	Raid struct {
		// ArrayName of the raid device, most often this will be /dev/md0 and so forth
		ArrayName string
		// Devices the devices to form a raid device
		Devices []string
		// Level the raidlevel to use, can be one of 0,1
		Level RaidLevel
		// CreateOptions required during raid creation, example: --metadata=1.0 for uefi boot partition
		CreateOptions []string
		// Spares defaults to 0
		Spares int
	}

	// VolumeGroup is optional, if given the devices must match.
	VolumeGroup struct {
		// Name of the volumegroup without the /dev prefix
		Name string
		// Devices the devices to form a volumegroup device
		Devices []string
		// Tags to attach to the volumegroup
		Tags []string
	}

	// LogicalVolume is a block devices created with lvm on top of a volumegroup
	LogicalVolume struct {
		// Name the name of the logical volume, without /dev prefix, will be accessible at /dev/vgname/lvname
		Name string
		// VolumeGroup the name of the volumegroup
		VolumeGroup string
		// Size of this LV in mebibytes (MiB)
		Size uint64
		// LVMType can be either striped or raid1
		LVMType LVMType
	}

	// DiskPartition is a single partition on a device, only GPT partition types are supported
	// FIXME overlaps with DiskPartition in machine.go which is part of reinstall feature
	DiskPartition2 struct {
		// Number of this partition, will be added to partitionprefix
		Number uint8
		// Label to enhance readability
		Label *string
		// Size of this partition in mebibytes (MiB)
		// if "0" is given the rest of the device will be used, this requires Number to be the highest in this partition
		Size uint64
		// GPTType defines the GPT partition type
		GPTType *GPTType
	}
)

// Validate a existing FilesystemLayout
func (f *FilesystemLayout) Validate() error {
	// check device existence from disk.partition -> raid.device -> filesystem
	// collect all provided devices
	providedDevices := make(map[string]bool)
	for _, disk := range f.Disks {
		err := disk.validate()
		if err != nil {
			return err
		}
		providedDevices[disk.Device] = true
		for _, partition := range disk.Partitions {
			devname := fmt.Sprintf("%s%d", disk.PartitionPrefix, partition.Number)
			providedDevices[devname] = true
		}
	}

	// Raid should also be checked if devices are provided
	// Raidlevel must be in the supported range
	for _, raid := range f.Raid {
		for _, device := range raid.Devices {
			_, ok := providedDevices[device]
			if !ok {
				return fmt.Errorf("device:%s not provided by disk for raid:%s", device, raid.ArrayName)
			}
		}
		providedDevices[raid.ArrayName] = true

		_, ok := SupportedRaidLevels[raid.Level]
		if !ok {
			return fmt.Errorf("given raidlevel:%s is not supported", raid.Level)
		}
	}

	vgdevices := make(map[string]bool)
	// VolumeGroups may be on top of disks, partitions and raid devices
	for _, vg := range f.VolumeGroups {
		for _, device := range vg.Devices {
			_, ok := providedDevices[device]
			if !ok {
				return fmt.Errorf("device:%s not provided by machine for vg:%s", device, vg.Name)
			}
		}
		vgdevices[vg.Name] = true
	}

	// LogicalVolumes must be on top of volumegroups
	for _, lv := range f.LogicalVolumes {
		_, ok := vgdevices[lv.VolumeGroup]
		if !ok {
			return fmt.Errorf("volumegroup:%s not configured for lv:%s", lv.VolumeGroup, lv.Name)
		}
		providedDevices[path.Join("/dev/", lv.VolumeGroup, lv.Name)] = true
	}

	// check if all fs devices are provided
	// given format must be supported
	for _, fs := range f.Filesystems {
		if fs.Format == TMPFS {
			continue
		}
		_, ok := providedDevices[fs.Device]
		if !ok {
			return fmt.Errorf("device:%s for filesystem:%s is not configured", fs.Device, *fs.Path)
		}
		_, ok = SupportedFormats[fs.Format]
		if !ok {
			return fmt.Errorf("filesystem:%s format:%s is not supported", *fs.Path, fs.Format)
		}
	}

	// validate constraints
	err := f.Constraints.validate()
	if err != nil {
		return err
	}

	return nil
}

func (c *FilesystemLayoutConstraints) validate() error {
	// no pure wildcard in images
	for os, vc := range c.Images {
		if os == "*" {
			return fmt.Errorf("just '*' is not allowed as image os constraint")
		}
		// a single "*" is possible
		if strings.TrimSpace(vc) == "*" {
			continue
		}
		_, _, err := convertToOpAndVersion(vc)
		if err != nil {
			return err
		}

	}
	// no wildcard in size
	for _, s := range c.Sizes {
		if strings.Contains(s, "*") {
			return fmt.Errorf("no wildcard allowed in size constraint")
		}
	}
	return nil
}

var validOPS = map[string]bool{"=": true, "!=": true, ">": true, "<": true, ">=": true, "=>": true, "<=": true, "=<": true, "~": true, "~>": true, "^": true}

func convertToOpAndVersion(versionconstraint string) (string, *semver.Version, error) {
	// a version constrain op is given it must be seperated by a whitespace
	parts := strings.SplitN(versionconstraint, " ", 2)
	// might be a single specific version, then it must parse into a semver
	if len(parts) == 1 {
		version, err := semver.NewVersion(parts[0])
		if err != nil {
			return "", nil, fmt.Errorf("given imageconstraint:%s is not valid, missing space between op and version? %w", parts[0], err)
		}
		return "", version, nil
	}
	if len(parts) >= 2 {
		op := parts[0]
		_, ok := validOPS[op]
		if !ok {
			return "", nil, fmt.Errorf("given imageconstraint op:%s is not supported", op)
		}
		version, err := semver.NewVersion(parts[1])
		if err != nil {
			return "", nil, fmt.Errorf("given version:%s is not valid:%w", parts[1], err)
		}
		return op, version, nil
	}
	return "", nil, fmt.Errorf("could not find a valid op or version in:%s", versionconstraint)
}

// Validate ensures that for all Filesystemlayouts not more than one constraint matches the same size and image constraint
func (fls FilesystemLayouts) Validate() error {
	var allConstraints []FilesystemLayoutConstraints
	for _, fl := range fls {
		allConstraints = append(allConstraints, fl.Constraints)
	}

	sizeAndImageOSToConstraint := make(map[string][]string)
	for _, c := range allConstraints {
		// if both size and image is empty, overlapping is possible because to be able to develop layouts
		if len(c.Sizes) == 0 && len(c.Images) == 0 {
			continue
		}
		for _, s := range c.Sizes {
			for os, versionConstraint := range c.Images {
				sizeAndImageOS := s + os
				versionConstraints, ok := sizeAndImageOSToConstraint[sizeAndImageOS]
				if !ok {
					sizeAndImageOSToConstraint[sizeAndImageOS] = []string{}
				}
				versionConstraints = append(versionConstraints, versionConstraint)

				err := hasCollisions(versionConstraints)
				if err != nil {
					return fmt.Errorf("combination of size:%s and image: %s %s already exists", s, os, versionConstraint)
				}

				sizeAndImageOSToConstraint[sizeAndImageOS] = versionConstraints
			}
		}
	}
	return nil
}

func hasCollisions(versionConstraints []string) error {
	// simple exclusion
	for _, vc := range versionConstraints {
		if strings.TrimSpace(vc) == "*" && len(versionConstraints) > 1 {
			return fmt.Errorf("at least one `*` and more than one constraint")
		}
	}

	for _, vci := range versionConstraints {
		for _, vcj := range versionConstraints {
			if vci == vcj {
				continue
			}
			constrainti, err := semver.NewConstraint(vci)
			if err != nil {
				return err
			}
			constraintj, err := semver.NewConstraint(vcj)
			if err != nil {
				return err
			}
			_, versioni, err := convertToOpAndVersion(vci)
			if err != nil {
				return err
			}
			_, versionj, err := convertToOpAndVersion(vcj)
			if err != nil {
				return err
			}
			if constrainti.Check(versionj) {
				return fmt.Errorf("constraint:%s overlaps:%s", constrainti, constraintj)
			}
			if constraintj.Check(versioni) {
				return fmt.Errorf("constraint:%s overlaps:%s", constraintj, constrainti)
			}
		}
	}
	return nil
}

// validate disk for
// - variable sized partition must be the last
// - GPTType is supported
func (d Disk) validate() error {
	partNumbers := make(map[uint8]bool)
	parts := make([]uint64, len(d.Partitions)+1)
	hasVariablePartition := false
	for _, partition := range d.Partitions {
		if partition.Size == 0 {
			hasVariablePartition = true
		}
		parts[partition.Number] = partition.Size

		_, ok := partNumbers[partition.Number]
		if ok {
			return fmt.Errorf("device:%s partition number:%d given more than once", d.Device, partition.Number)
		}

		partNumbers[partition.Number] = true

		if partition.GPTType != nil {
			_, ok := SupportedGPTTypes[*partition.GPTType]
			if !ok {
				return fmt.Errorf("given GPTType:%s for partition:%d on disk:%s is not supported", *partition.GPTType, partition.Number, d.Device)
			}
		}
	}
	if hasVariablePartition && (parts[len(parts)-1] != 0) {
		return fmt.Errorf("device:%s variable sized partition not the last one", d.Device)
	}
	return nil
}

// matches decides if for given size and image the constraints will match
func (c *FilesystemLayoutConstraints) matches(sizeID, imageID string) bool {
	_, ok := sizeMap(c.Sizes)[sizeID]
	if !ok {
		return false
	}
	// Size matches
	for os, versionconstraint := range c.Images {
		imageos, version, err := utils.GetOsAndSemverFromImage(imageID)
		if err != nil {
			return false
		}
		if os != imageos {
			continue
		}
		c, err := semver.NewConstraint(versionconstraint)
		if err != nil {
			return false
		}
		if c.Check(version) {
			return true
		}
	}

	return false
}

// From will pick a filesystemlayout from all filesystemlayouts which matches given size and image
func (fls FilesystemLayouts) From(size, image string) (*FilesystemLayout, error) {
	for _, fl := range fls {
		if fl.Constraints.matches(size, image) {
			return &fl, nil
		}
	}
	return nil, fmt.Errorf("could not find a matching filesystemLayout for size:%s and image:%s", size, image)
}

// Matches the specific FilesystemLayout against the selected Hardware
func (fl *FilesystemLayout) Matches(hardware MachineHardware) error {
	requiredDevices := make(map[string]uint64)
	existingDevices := make(map[string]uint64)
	for _, disk := range fl.Disks {
		var requiredSize uint64
		for _, partition := range disk.Partitions {
			requiredSize += partition.Size
		}
		requiredDevices[disk.Device] = requiredSize
	}

	for _, disk := range hardware.Disks {
		diskName := disk.Name
		if !strings.HasPrefix(diskName, "/dev/") {
			diskName = fmt.Sprintf("/dev/%s", disk.Name)
		}
		// convert bytes to mebibytes
		size := disk.Size / (1024 * 1024)
		existingDevices[diskName] = size
	}

	for requiredDevice, requiredSize := range requiredDevices {
		existingSize, ok := existingDevices[requiredDevice]
		if !ok {
			return fmt.Errorf("device:%s does not exist on given hardware", requiredDevice)
		}
		if existingSize < requiredSize {
			return fmt.Errorf("device:%s is not big enough required:%dMiB, existing:%dMiB", requiredDevice, requiredSize, existingSize)
		}
	}
	return nil
}

func ToFormat(format string) (*Format, error) {
	f := Format(format)
	_, ok := SupportedFormats[f]
	if !ok {
		return nil, fmt.Errorf("given format:%s is not supported", format)
	}
	return &f, nil
}

func ToGPTType(gptType string) (*GPTType, error) {
	g := GPTType(gptType)
	_, ok := SupportedGPTTypes[g]
	if !ok {
		return nil, fmt.Errorf("given GPTType:%s is not supported", gptType)
	}
	return &g, nil
}

func ToRaidLevel(level string) (*RaidLevel, error) {
	l := RaidLevel(level)
	_, ok := SupportedRaidLevels[l]
	if !ok {
		return nil, fmt.Errorf("given raidlevel:%s is not supported", level)
	}
	return &l, nil
}

func sizeMap(sizes []string) map[string]bool {
	sm := make(map[string]bool)
	for _, s := range sizes {
		sm[s] = true
	}
	return sm
}
