package v1

import "github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"

type (
	FilesystemLayoutBase struct {
		Filesystems    []Filesystem                `json:"filesystems" description:"list of filesystems to create" optional:"true"`
		Disks          []Disk                      `json:"disks" description:"list of disks that belong to this layout" optional:"true"`
		Raid           []Raid                      `json:"raid" description:"list of raid arrays to create" optional:"true"`
		VolumeGroups   []VolumeGroup               `json:"volumegroups" description:"list of volumegroups to create" optional:"true"`
		LogicalVolumes []LogicalVolume             `json:"logicalvolumes" description:"list of logicalvolumes to create" optional:"true"`
		Constraints    FilesystemLayoutConstraints `json:"constraints" description:"constraints which must match that this layout is taken, if sizes and images are empty these are develop layouts"`
	}
	FilesystemLayoutResponse struct {
		Common
		FilesystemLayoutBase
	}

	FilesystemLayoutCreateRequest struct {
		Common
		FilesystemLayoutBase
	}

	FilesystemLayoutUpdateRequest struct {
		Common
		FilesystemLayoutBase
	}

	FilesystemLayoutTryRequest struct {
		Size  string `json:"size" description:"machine size to try"`
		Image string `json:"image" description:"image to try"`
	}

	FilesystemLayoutMatchRequest struct {
		Machine          string `json:"machine" description:"machine id to check"`
		FilesystemLayout string `json:"filesystemlayout" description:"filesystemlayout id to check"`
	}

	FilesystemLayoutConstraints struct {
		Sizes  []string          `json:"sizes" description:"list of sizes this layout applies to" optional:"true"`
		Images map[string]string `json:"images" description:"list of images this layout applies to"`
	}
	Filesystem struct {
		Path          *string  `json:"path" description:"the mountpoint where this filesystem should be mounted on" optional:"true"`
		Device        string   `json:"device" description:"the underlaying device where this filesystem should be created"`
		Format        string   `json:"format" description:"the filesystem format"`
		Label         *string  `json:"label" description:"optional label for this this filesystem" optional:"true"`
		MountOptions  []string `json:"mountoptions" description:"the options to use to mount this filesystem" optional:"true"`
		CreateOptions []string `json:"createoptions" description:"the options to use to create (mkfs) this filesystem" optional:"true"`
	}
	Disk struct {
		Device          string          `json:"device" description:"the device to create the partitions"`
		Partitions      []DiskPartition `json:"partitions" description:"list of partitions to create on this disk" optional:"true"`
		WipeOnReinstall bool            `json:"wipeonreinstall" description:"if set to true, this disk will be wiped before reinstallation"`
	}
	Raid struct {
		ArrayName     string   `json:"arrayname" description:"the name of the resulting array device"`
		Devices       []string `json:"devices" description:"list of devices to form the raid array from" optional:"true"`
		Level         string   `json:"level" description:"raid level to create, should be 0 or 1"`
		CreateOptions []string `json:"createoptions" description:"the options to use to create the raid array" optional:"true"`
		Spares        int      `json:"spares" description:"number of spares for the raid array"`
	}
	DiskPartition struct {
		Number  uint8   `json:"number" description:"partition number, will be appended to partitionprefix to create the final devicename"`
		Label   *string `json:"label" description:"optional label for this this partition" optional:"true"`
		Size    uint64  `json:"size" description:"size in mebibytes (MiB) of this partition"`
		GPTType *string `json:"gpttype" description:"the gpt partition table type of this partition"`
	}
	VolumeGroup struct {
		Name    string   `json:"name" description:"the name of the resulting volume group"`
		Devices []string `json:"devices" description:"list of devices to form the volume group from" optional:"true"`
		Tags    []string `json:"tags" description:"list of tags to add to the volume group" optional:"true"`
	}

	LogicalVolume struct {
		Name        string `json:"name" description:"the name of the logical volume"`
		VolumeGroup string `json:"volumegroup" description:"the name of the volume group where to create the logical volume onto"`
		Size        uint64 `json:"size" description:"size in mebibytes (MiB) of this volume"`
		LVMType     string `json:"lvmtype" description:"the type of this logical volume can be either linear|striped|raid1"`
	}
)

func NewFilesystemLayout(f FilesystemLayoutCreateRequest) (*metal.FilesystemLayout, error) {
	var (
		fss = []metal.Filesystem{}
		ds  = []metal.Disk{}
		rs  = []metal.Raid{}
		vgs = []metal.VolumeGroup{}
		lvs = []metal.LogicalVolume{}
	)
	for _, fs := range f.Filesystems {
		format, err := metal.ToFormat(fs.Format)
		if err != nil {
			return nil, err
		}
		v1fs := metal.Filesystem{
			Path:          fs.Path,
			Device:        string(fs.Device),
			Format:        *format,
			Label:         fs.Label,
			MountOptions:  fs.MountOptions,
			CreateOptions: fs.CreateOptions,
		}
		fss = append(fss, v1fs)
	}
	for _, disk := range f.Disks {
		parts := []metal.DiskPartition{}
		for _, p := range disk.Partitions {
			part := metal.DiskPartition{
				Number: p.Number,
				Size:   p.Size,
				Label:  p.Label,
			}
			if p.GPTType != nil {
				gptType, err := metal.ToGPTType(*p.GPTType)
				if err != nil {
					return nil, err
				}
				part.GPTType = gptType
			}
			parts = append(parts, part)
		}
		d := metal.Disk{
			Device:          string(disk.Device),
			Partitions:      parts,
			WipeOnReinstall: disk.WipeOnReinstall,
		}
		ds = append(ds, d)
	}
	for _, raid := range f.Raid {
		level, err := metal.ToRaidLevel(raid.Level)
		if err != nil {
			return nil, err
		}
		r := metal.Raid{
			ArrayName:     raid.ArrayName,
			Devices:       raid.Devices,
			Level:         *level,
			CreateOptions: raid.CreateOptions,
			Spares:        raid.Spares,
		}
		rs = append(rs, r)
	}
	for _, v := range f.VolumeGroups {
		vg := metal.VolumeGroup{
			Name:    v.Name,
			Devices: v.Devices,
			Tags:    v.Tags,
		}
		vgs = append(vgs, vg)
	}
	for _, l := range f.LogicalVolumes {
		lvmtype, err := metal.ToLVMType(l.LVMType)
		if err != nil {
			return nil, err
		}
		lv := metal.LogicalVolume{
			Name:        l.Name,
			VolumeGroup: l.VolumeGroup,
			Size:        l.Size,
			LVMType:     *lvmtype,
		}
		lvs = append(lvs, lv)
	}
	fl := &metal.FilesystemLayout{
		Base: metal.Base{
			ID: f.ID,
		},
		Filesystems:    fss,
		Disks:          ds,
		Raid:           rs,
		VolumeGroups:   vgs,
		LogicalVolumes: lvs,
		Constraints: metal.FilesystemLayoutConstraints{
			Sizes:  f.Constraints.Sizes,
			Images: f.Constraints.Images,
		},
	}
	if f.Name != nil {
		fl.Name = *f.Name
	}
	if f.Description != nil {
		fl.Description = *f.Description
	}
	return fl, nil
}

func NewFilesystemLayoutResponse(f *metal.FilesystemLayout) *FilesystemLayoutResponse {
	if f == nil {
		return nil
	}
	var (
		fss []Filesystem
		ds  []Disk
		rs  []Raid
		vgs []VolumeGroup
		lvs []LogicalVolume
	)
	for _, fs := range f.Filesystems {
		v1fs := Filesystem{
			Path:          fs.Path,
			Device:        string(fs.Device),
			Format:        string(fs.Format),
			Label:         fs.Label,
			MountOptions:  fs.MountOptions,
			CreateOptions: fs.CreateOptions,
		}
		fss = append(fss, v1fs)
	}
	for _, disk := range f.Disks {
		var parts []DiskPartition
		for _, p := range disk.Partitions {
			part := DiskPartition{
				Number:  p.Number,
				Size:    p.Size,
				Label:   p.Label,
				GPTType: (*string)(p.GPTType),
			}
			parts = append(parts, part)
		}
		d := Disk{
			Device:          string(disk.Device),
			Partitions:      parts,
			WipeOnReinstall: disk.WipeOnReinstall,
		}
		ds = append(ds, d)
	}
	for _, raid := range f.Raid {
		r := Raid{
			ArrayName:     raid.ArrayName,
			Devices:       raid.Devices,
			Level:         string(raid.Level),
			CreateOptions: raid.CreateOptions,
			Spares:        raid.Spares,
		}
		rs = append(rs, r)
	}
	for _, v := range f.VolumeGroups {
		vg := VolumeGroup{
			Name:    v.Name,
			Devices: v.Devices,
			Tags:    v.Tags,
		}
		vgs = append(vgs, vg)
	}
	for _, l := range f.LogicalVolumes {
		lv := LogicalVolume{
			Name:        l.Name,
			VolumeGroup: l.VolumeGroup,
			Size:        l.Size,
			LVMType:     string(l.LVMType),
		}
		lvs = append(lvs, lv)
	}
	flr := &FilesystemLayoutResponse{
		Common: Common{
			Identifiable: Identifiable{
				ID: f.ID,
			},
			Describable: Describable{
				Name:        &f.Name,
				Description: &f.Description,
			},
		},
		FilesystemLayoutBase: FilesystemLayoutBase{
			Filesystems:    fss,
			Disks:          ds,
			Raid:           rs,
			VolumeGroups:   vgs,
			LogicalVolumes: lvs,
			Constraints: FilesystemLayoutConstraints{
				Sizes:  f.Constraints.Sizes,
				Images: f.Constraints.Images,
			},
		},
	}
	return flr
}
