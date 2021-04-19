package v1

import "github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"

type (
	FilesystemLayoutResponse struct {
		Common
		Filesystems []Filesystem
		Disks       []Disk
		Raid        []Raid
		Constraints FilesystemLayoutConstraints
	}

	FilesystemLayoutCreateRequest struct {
		Common
		Filesystems []Filesystem
		Disks       []Disk
		Raid        []Raid
		Constraints FilesystemLayoutConstraints
	}

	FilesystemLayoutUpdateRequest struct {
		Common
		Filesystems []Filesystem
		Disks       []Disk
		Raid        []Raid
		Constraints FilesystemLayoutConstraints
	}

	FilesystemLayoutConstraints struct {
		Sizes  []string
		Images []string
	}
	Filesystem struct {
		Path         *string
		Device       string
		Format       string
		Label        *string
		MountOptions []string
		Options      []string
	}
	Disk struct {
		Device          string
		PartitionPrefix string
		Partitions      []DiskPartition
		Wipe            bool
	}
	Raid struct {
		Name    string
		Devices []string
		Level   string
		Options []string
		Spares  int
	}
	DiskPartition struct {
		Number  uint8
		Label   *string
		Size    int64
		GUID    *string
		GPTType *string
	}
)

func NewFilesystemLayout(f FilesystemLayoutCreateRequest) (*metal.FilesystemLayout, error) {
	var (
		fss []metal.Filesystem
		ds  []metal.Disk
		rs  []metal.Raid
	)
	for _, fs := range f.Filesystems {
		format, err := metal.ToFormat(fs.Format)
		if err != nil {
			return nil, err
		}
		v1fs := metal.Filesystem{
			Path:         fs.Path,
			Device:       string(fs.Device),
			Format:       *format,
			Label:        fs.Label,
			MountOptions: fs.MountOptions,
			Options:      fs.Options,
		}
		fss = append(fss, v1fs)
	}
	for _, disk := range f.Disks {
		var parts []metal.DiskPartition2
		for _, p := range disk.Partitions {
			part := metal.DiskPartition2{
				Number: p.Number,
				Size:   p.Size,
				Label:  p.Label,
				GUID:   p.GUID,
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
			PartitionPrefix: disk.PartitionPrefix,
			Partitions:      parts,
			Wipe:            disk.Wipe,
		}
		ds = append(ds, d)
	}
	for _, raid := range f.Raid {
		level, err := metal.ToRaidLevel(raid.Level)
		if err != nil {
			return nil, err
		}
		r := metal.Raid{
			Name:    raid.Name,
			Devices: raid.Devices,
			Level:   *level,
			Options: raid.Options,
			Spares:  raid.Spares,
		}
		rs = append(rs, r)
	}
	fl := &metal.FilesystemLayout{
		Base: metal.Base{
			ID: f.ID,
		},
		Filesystems: fss,
		Disks:       ds,
		Raid:        rs,
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
	)
	for _, fs := range f.Filesystems {
		v1fs := Filesystem{
			Path:         fs.Path,
			Device:       string(fs.Device),
			Format:       string(fs.Format),
			Label:        fs.Label,
			MountOptions: fs.MountOptions,
			Options:      fs.Options,
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
				GUID:    (*string)(p.GUID),
				GPTType: (*string)(p.GPTType),
			}
			parts = append(parts, part)
		}
		d := Disk{
			Device:          string(disk.Device),
			PartitionPrefix: disk.PartitionPrefix,
			Partitions:      parts,
			Wipe:            disk.Wipe,
		}
		ds = append(ds, d)
	}
	for _, raid := range f.Raid {
		r := Raid{
			Name:    raid.Name,
			Devices: raid.Devices,
			Level:   string(raid.Level),
			Options: raid.Options,
			Spares:  raid.Spares,
		}
		rs = append(rs, r)
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
		Filesystems: fss,
		Disks:       ds,
		Raid:        rs,
		Constraints: FilesystemLayoutConstraints{
			Sizes:  f.Constraints.Sizes,
			Images: f.Constraints.Images,
		},
	}
	return flr
}
