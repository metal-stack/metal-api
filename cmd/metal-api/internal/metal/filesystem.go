package metal

import (
	"errors"
	"fmt"
	"path"
	"sort"
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

	// LVMTypeLinear append across all physical volumes
	LVMTypeLinear = LVMType("linear")
	// LVMTypeStriped stripe across all physical volumes
	LVMTypeStriped = LVMType("striped")
	// LVMTypeStripe mirror with raid across all physical volumes
	LVMTypeRaid1 = LVMType("raid1")
)

var (
	SupportedFormats    = map[Format]bool{VFAT: true, EXT3: true, EXT4: true, SWAP: true, TMPFS: true, NONE: true}
	SupportedGPTTypes   = map[GPTType]bool{GPTBoot: true, GPTLinux: true, GPTLinuxLVM: true, GPTLinuxRaid: true}
	SupportedRaidLevels = map[RaidLevel]bool{RaidLevel0: true, RaidLevel1: true}
	SupportedLVMTypes   = map[LVMType]bool{LVMTypeLinear: true, LVMTypeStriped: true, LVMTypeRaid1: true}
)

type (
	// FilesystemLayouts is a slice of FilesystemLayout
	FilesystemLayouts []FilesystemLayout
	// FilesystemLayout to be created on the given machine
	FilesystemLayout struct {
		Base
		// Filesystems to create on the server
		Filesystems []Filesystem `rethinkdb:"filesystems" json:"filesystem"`
		// Disks to configure in the server with their partitions
		Disks []Disk `rethinkdb:"disks" json:"disks"`
		// Raid if not empty, create raid arrays out of the individual disks, to place filesystems onto
		Raid []Raid `rethinkdb:"raid" json:"raid"`
		// VolumeGroups to create
		VolumeGroups []VolumeGroup `rethinkdb:"volumegroups" json:"volumegroups"`
		// LogicalVolumes to create on top of VolumeGroups
		LogicalVolumes LogicalVolumes `rethinkdb:"logicalvolumes" json:"logicalvolumes"`
		// Constraints which must match to select this Layout
		Constraints FilesystemLayoutConstraints `rethinkdb:"constraints" json:"constraints"`
	}

	// LogicalVolumes is a slice of LogicalVolume
	LogicalVolumes []LogicalVolume

	FilesystemLayoutConstraints struct {
		// Sizes defines the list of sizes this layout applies to
		Sizes []string `rethinkdb:"sizes" json:"sizes"`
		// Images defines a map from os to versionconstraint
		// the combination of os and versionconstraint per size must be conflict free over all filesystemlayouts
		Images map[string]string `rethinkdb:"images" json:"images"`
	}

	RaidLevel string
	Format    string
	GPTType   string
	LVMType   string

	// Filesystem defines a single filesystem to be mounted
	Filesystem struct {
		// Path defines the mountpoint, if nil, it will not be mounted
		Path *string `rethinkdb:"path" json:"path"`
		// Device where the filesystem is created on, must be the full device path seen by the OS
		Device string `rethinkdb:"device" json:"device"`
		// Format is the type of filesystem should be created
		Format Format `rethinkdb:"format" json:"format"`
		// Label is optional enhances readability
		Label *string `rethinkdb:"label" json:"label"`
		// MountOptions which might be required
		MountOptions []string `rethinkdb:"mountoptions" json:"mountoptions"`
		// CreateOptions during filesystem creation
		CreateOptions []string `rethinkdb:"createoptions" json:"createoptions"`
	}

	// Disk represents a single block device visible from the OS, required
	Disk struct {
		// Device is the full device path
		Device string `rethinkdb:"device" json:"device"`
		// Partitions to create on this device
		Partitions []DiskPartition `rethinkdb:"partitions" json:"partitions"`
		// WipeOnReinstall, if set to true the whole disk will be erased if reinstall happens
		// during fresh install all disks are wiped
		WipeOnReinstall bool `rethinkdb:"wipeonreinstall" json:"wipeonreinstall"`
	}

	// Raid is optional, if given the devices must match.
	Raid struct {
		// ArrayName of the raid device, most often this will be /dev/md0 and so forth
		ArrayName string `rethinkdb:"arrayname" json:"arrayname"`
		// Devices the devices to form a raid device
		Devices []string `rethinkdb:"devices" json:"devices"`
		// Level the raidlevel to use, can be one of 0,1
		Level RaidLevel `rethinkdb:"raidlevel" json:"raidlevel"`
		// CreateOptions required during raid creation, example: --metadata=1.0 for uefi boot partition
		CreateOptions []string `rethinkdb:"createoptions" json:"createoptions"`
		// Spares defaults to 0
		Spares int `rethinkdb:"spares" json:"spares"`
	}

	// VolumeGroup is optional, if given the devices must match.
	VolumeGroup struct {
		// Name of the volumegroup without the /dev prefix
		Name string `rethinkdb:"name" json:"name"`
		// Devices the devices to form a volumegroup device
		Devices []string `rethinkdb:"devices" json:"devices"`
		// Tags to attach to the volumegroup
		Tags []string `rethinkdb:"tags" json:"tags"`
	}

	// LogicalVolume is a block devices created with lvm on top of a volumegroup
	LogicalVolume struct {
		// Name the name of the logical volume, without /dev prefix, will be accessible at /dev/vgname/lvname
		Name string `rethinkdb:"name" json:"name"`
		// VolumeGroup the name of the volumegroup
		VolumeGroup string `rethinkdb:"volumegroup" json:"volumegroup"`
		// Size of this LV in mebibytes (MiB), if zero all remaining space in the vg will be used.
		Size uint64 `rethinkdb:"size" json:"size"`
		// LVMType can be either linear, striped or raid1
		LVMType LVMType `rethinkdb:"lvmtype" json:"lvmtype"`
	}

	// DiskPartition is a single partition on a device, only GPT partition types are supported
	DiskPartition struct {
		// Number of this partition, will be added to partitionprefix
		Number uint8 `rethinkdb:"number" json:"number"`
		// Label to enhance readability
		Label *string `rethinkdb:"label" json:"label"`
		// Size of this partition in mebibytes (MiB)
		// if "0" is given the rest of the device will be used, this requires Number to be the highest in this partition
		Size uint64 `rethinkdb:"size" json:"size"`
		// GPTType defines the GPT partition type
		GPTType *GPTType `rethinkdb:"gpttype" json:"gpttype"`
	}
)

// Validate a existing FilesystemLayout
func (f FilesystemLayout) Validate() error {
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
			partitionPrefix := ""
			if strings.HasPrefix(disk.Device, "/dev/nvme") {
				partitionPrefix = "p"
			}
			devname := fmt.Sprintf("%s%s%d", disk.Device, partitionPrefix, partition.Number)
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

	vgdevices := make(map[string]int)
	// VolumeGroups may be on top of disks, partitions and raid devices
	for _, vg := range f.VolumeGroups {
		for _, device := range vg.Devices {
			_, ok := providedDevices[device]
			if !ok {
				return fmt.Errorf("device:%s not provided by machine for vg:%s", device, vg.Name)
			}
		}
		vgdevices[vg.Name] = len(vg.Devices)
	}

	// LogicalVolumes must be on top of volumegroups
	err := f.LogicalVolumes.validate()
	if err != nil {
		return err
	}
	for _, lv := range f.LogicalVolumes {
		_, ok := vgdevices[lv.VolumeGroup]
		if !ok {
			return fmt.Errorf("volumegroup:%s not configured for lv:%s", lv.VolumeGroup, lv.Name)
		}
		// raid or striped lvmtype is only possible for more than one disk
		if lv.LVMType == LVMTypeRaid1 || lv.LVMType == LVMTypeStriped {
			if vgdevices[lv.VolumeGroup] < 2 {
				return fmt.Errorf("fsl:%q lv:%s in vg:%s is configured for lvmtype:%s but has only %d disk, consider linear instead", f.ID, lv.Name, lv.VolumeGroup, lv.LVMType, vgdevices[lv.VolumeGroup])
			}
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
		err := validateCreateOptions(fs.CreateOptions)
		if err != nil {
			return err
		}
	}

	// validate constraints
	err = f.Constraints.validate()
	if err != nil {
		return err
	}

	return nil
}

func validateCreateOptions(opts []string) error {
	var errs []error
	for _, opt := range opts {
		if len(strings.Fields(opt)) > 1 {
			errs = append(errs, fmt.Errorf("the given createoption:%q contains whitespace and must be split into separate options", opt))
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
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

	sizeSet := make(map[string]bool)
	// no wildcard in size
	for _, s := range c.Sizes {
		if strings.Contains(s, "*") {
			return fmt.Errorf("no wildcard allowed in size constraint")
		}
		_, ok := sizeSet[s]
		if !ok {
			sizeSet[s] = true
		} else {
			return fmt.Errorf("size %s is configured more than once", s)
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
	allConstraints := make(map[string]FilesystemLayoutConstraints)
	for _, fl := range fls {
		allConstraints[fl.ID] = fl.Constraints
	}

	violations := []string{}
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
					violations = append(violations, fmt.Sprintf("%s->[%s %s]", s, os, versionConstraint))
				}

				sizeAndImageOSToConstraint[sizeAndImageOS] = versionConstraints
			}
		}
	}
	if len(violations) > 0 {
		return fmt.Errorf("these combinations already exist:%s", strings.Join(violations, ","))
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
			if constrainti.Check(versionj) && constraintj.Check(versioni) {
				return fmt.Errorf("constraint:%s overlaps:%s", constrainti, constraintj)
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

// validate logicalvolume
// - variable sized lv must be the last
func (lvms LogicalVolumes) validate() error {
	if len(lvms) == 0 {
		return nil
	}

	for i, lvm := range lvms {
		if lvm.Size == 0 && i != len(lvms)-1 {
			return fmt.Errorf("lv:%s in vg:%s, variable sized lv must be the last", lvm.Name, lvm.VolumeGroup)
		}
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

// IsReinstallable returns true if at least one disk configures has WipeOnReInstall set, otherwise false
func (fl *FilesystemLayout) IsReinstallable() bool {
	for _, d := range fl.Disks {
		if d.WipeOnReinstall {
			return true
		}
	}
	return false
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

func supportedFormats() string {
	sf := []string{}
	for f := range SupportedFormats {
		sf = append(sf, string(f))
	}
	sort.Strings(sf)
	return strings.Join(sf, ",")
}
func supportedGPTTypes() string {
	sf := []string{}
	for f := range SupportedGPTTypes {
		sf = append(sf, string(f))
	}
	sort.Strings(sf)
	return strings.Join(sf, ",")
}
func supportedRaidLevels() string {
	sf := []string{}
	for f := range SupportedRaidLevels {
		sf = append(sf, string(f))
	}
	sort.Strings(sf)
	return strings.Join(sf, ",")
}
func supportedLVMTypes() string {
	sf := []string{}
	for f := range SupportedLVMTypes {
		sf = append(sf, string(f))
	}
	sort.Strings(sf)
	return strings.Join(sf, ",")
}

func ToFormat(format string) (*Format, error) {
	f := Format(format)
	_, ok := SupportedFormats[f]
	if !ok {
		return nil, fmt.Errorf("given format:%s is not supported, but:%s", format, supportedFormats())
	}
	return &f, nil
}

func ToGPTType(gptType string) (*GPTType, error) {
	g := GPTType(gptType)
	_, ok := SupportedGPTTypes[g]
	if !ok {
		return nil, fmt.Errorf("given GPTType:%s is not supported, but:%s", gptType, supportedGPTTypes())
	}
	return &g, nil
}

func ToRaidLevel(level string) (*RaidLevel, error) {
	l := RaidLevel(level)
	_, ok := SupportedRaidLevels[l]
	if !ok {
		return nil, fmt.Errorf("given raidlevel:%s is not supported, but:%s", level, supportedRaidLevels())
	}
	return &l, nil
}

func ToLVMType(lvmtype string) (*LVMType, error) {
	l := LVMType(lvmtype)
	_, ok := SupportedLVMTypes[l]
	if !ok {
		return nil, fmt.Errorf("given lvmtype:%s is not supported, but:%s", lvmtype, supportedLVMTypes())
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
