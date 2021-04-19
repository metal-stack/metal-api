package metal

import (
	"fmt"
	"path/filepath"
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
	// EFISystemPartition see https://en.wikipedia.org/wiki/EFI_system_partition
	EFISystemPartition = GUID("C12A7328-F81F-11D2-BA4B-00A0C93EC93B")

	// RaidLevel0 is a stripe of two or more disks
	RaidLevel0 = RaidLevel("0")
	// RaidLevel1 is a mirror of two disks
	RaidLevel1 = RaidLevel("1")
)

var (
	SupportedFormats    = []Format{VFAT, EXT3, EXT4, SWAP, NONE}
	SupportedGPTTypes   = []GPTType{GPTBoot, GPTLinux, GPTLinuxLVM, GPTLinuxRaid}
	SupportedGUIDs      = []GUID{EFISystemPartition}
	SupportedRaidLevels = []RaidLevel{RaidLevel0, RaidLevel1}
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
		// Constraints which must match to select this Layout
		Constraints FilesystemLayoutConstraints
	}

	FilesystemLayoutConstraints struct {
		// Sizes defines the list of sizes this layout applies to
		Sizes []string
		// Images defines a list of image glob patterns this layout should apply
		// the most specific combination of sizes and images will be picked fo a allocation
		Images []string
	}

	FilesystemOption string
	MountOption      string
	RaidOption       string
	RaidLevel        string
	Device           string
	Format           string
	GUID             string
	GPTType          string

	// Filesystem defines a single filesystem to be mounted
	Filesystem struct {
		// Path defines the mountpoint, if nil, it will not be mounted
		Path *string
		// Device where the filesystem is created on, must be the full device path seen by the OS
		Device Device
		// Format is the type of filesystem should be created
		Format Format
		// Label is optional enhances readability
		Label *string
		// MountOptions which might be required
		MountOptions []MountOption
		// Options during filesystem creation
		Options []FilesystemOption
	}

	// Disk represents a single block device visible from the OS, required
	Disk struct {
		// Device is the full device path
		Device Device
		// PartitionPrefix specifies which prefix is used if device is partitioned
		// e.g. device /dev/sda, first partition will be /dev/sda1, prefix is therefore /dev/sda
		// for nvme drives this is different, the prefix there is typically /dev/nvme0n1p
		PartitionPrefix string
		// Partitions to create on this device
		Partitions []DiskPartition2
		// Wipe, if set to true the partition table will be erase before new partitions will be created
		Wipe bool
	}

	// Raid is optional, if given the devices must match.
	// TODO inherit GPTType from underlay device ?
	Raid struct {
		// Name of the raid device, most often this will be /dev/md0 and so forth
		Name string
		// Devices the devices to form a raid device
		Devices []Device
		// Level the raidlevel to use, can be one of 0,1
		Level RaidLevel
		// Options required during raid creation, example: --metadata=1.0 for uefi boot partition
		Options []RaidOption
		// Spares defaults to 0
		Spares int
	}

	// DiskPartition is a single partition on a device, only GPT partition types are supported
	// FIXME overlaps with DiskPartition in machine.go which is part of reinstall feature
	DiskPartition2 struct {
		// Number of this partition, will be added to partitionprefix
		Number uint8
		// Label to enhance readability
		Label *string
		// Size of this partition in bytes
		// if "-1" is given the rest of the device will be used, this requires Number to be the highest in this partition
		Size int64
		// GUID of this partition
		GUID *GUID
		// GPTType defines the GPT partition type
		GPTType *GPTType
	}
)

// Validate a existing FilesystemLayout
func (f *FilesystemLayout) Validate() (bool, error) {
	// check disk partitions for correct partition order of variable partition
	for _, disk := range f.Disks {
		err := disk.validate()
		if err != nil {
			return false, err
		}
	}

	// check device existence from disk.partition -> raid.device -> filesystem
	// collect all provided devices
	providedDevices := make(map[string]bool)
	for _, disk := range f.Disks {
		for _, partition := range disk.Partitions {
			devname := fmt.Sprintf("%s%d", disk.PartitionPrefix, partition.Number)
			providedDevices[devname] = true
		}
	}

	// Raid should also be checked if devices are provided
	for _, raid := range f.Raid {
		for _, device := range raid.Devices {
			_, ok := providedDevices[string(device)]
			if !ok {
				return false, fmt.Errorf("device:%s not provided by disk in raid:%s", device, raid.Name)
			}
		}
		providedDevices[raid.Name] = true
	}

	// check if all fs devices are provided
	for _, fs := range f.Filesystems {
		_, ok := providedDevices[string(fs.Device)]
		if !ok {
			return false, fmt.Errorf("device:%s for filesystem:%s is not configured as raid or device", fs.Device, *fs.Path)
		}
	}

	return true, nil
}

// validate disk for
// - variable sized partition must be the last
func (d Disk) validate() error {
	partNumbers := make(map[uint8]bool)
	parts := make([]int64, len(d.Partitions)+1)
	hasVariablePartition := false
	for _, partition := range d.Partitions {
		if partition.Size == -1 {
			hasVariablePartition = true
		}
		parts[partition.Number] = partition.Size

		_, ok := partNumbers[partition.Number]
		if ok {
			return fmt.Errorf("device:%s partition number:%d given more than once", d.Device, partition.Number)
		}

		partNumbers[partition.Number] = true
	}
	if hasVariablePartition && (parts[len(parts)-1] != -1) {
		return fmt.Errorf("device:%s variable sized partition not the last one", d.Device)
	}
	return nil
}

// Matches decides if for given size and image the constraints will match
func (c *FilesystemLayoutConstraints) Matches(size Size, image Image) bool {
	_, ok := sizeMap(c.Sizes)[size.ID]
	if !ok {
		return false
	}
	// Size matches
	for _, i := range sortImageGlobs(c.Images) {
		matches, err := filepath.Match(i, image.ID)
		if err != nil {
			return false
		}
		// Image matches
		if matches {
			return true
		}
	}

	return false
}

// From will pick a filesystemlayout from all filesystemlayouts which matches given size and image
func (fls FilesystemLayouts) From(size Size, image Image) (*FilesystemLayout, error) {
	for _, fl := range fls {
		if fl.Constraints.Matches(size, image) {
			return &fl, nil
		}
	}
	return nil, fmt.Errorf("could not find a matchin filesystemLayout for size:%s and image:%s", size.ID, image.ID)
}

// Matches the specific FilesystemLayout against the selected Hardware
func (fl *FilesystemLayout) Matches(hardware MachineHardware) (bool, error) {
	requiredDevices := make(map[string]int64)
	existingDevices := make(map[string]int64)
	for _, disk := range fl.Disks {
		var requiredSize int64
		for _, partition := range disk.Partitions {
			requiredSize += partition.Size
		}
		requiredDevices[string(disk.Device)] = requiredSize
	}

	for _, disk := range hardware.Disks {
		existingDevices[disk.Name] = int64(disk.Size)
	}

	for requiredDevice, requiredSize := range requiredDevices {
		existingSize, ok := existingDevices[requiredDevice]
		if !ok {
			return false, fmt.Errorf("device:%s does not exist on given hardware", requiredDevice)
		}
		if existingSize < requiredSize {
			return false, fmt.Errorf("device:%s is not big enough required:%d, existing:%d", requiredDevice, requiredSize, existingSize)
		}
	}
	return true, nil
}

// FIXME implement overlapping filesystemlayout detection
// FIXME implement check if selected machine hardware matches with selected filesystemlayout

func sortImageGlobs(globs []string) []string {
	var sorted []string
	appendStar := false
	for _, i := range globs {
		if i == "*" {
			appendStar = true
			continue
		}
		sorted = append(sorted, i)
	}
	if appendStar {
		sorted = append(sorted, "*")
	}
	return sorted
}

func sizeMap(sizes []string) map[string]bool {
	sm := make(map[string]bool)
	for _, s := range sizes {
		sm[s] = true
	}
	return sm
}
