package metal

import (
	"reflect"
	"testing"

	"github.com/Masterminds/semver/v3"
)

var (
	s1 = "c1-large-x86"
	s2 = "c1-xlarge-x86"
	s3 = "s3-large-x86"
	i1 = "debian-10"
	i2 = "debian-10.0.20210101"
	i3 = "firewall-2"
	i4 = "centos-7"

	GPTInvalid = GPTType("ff00")
)

func TestFilesystemLayoutConstraint_Matches(t *testing.T) {
	type constraints struct {
		Sizes  []string
		Images map[string]string
	}
	type args struct {
		size  string
		image string
	}
	tests := []struct {
		name string
		c    constraints
		args args
		want bool
	}{
		{
			name: "default layout",
			c: constraints{
				Sizes:  []string{"c1-large-x86", "c1-xlarge-x86"},
				Images: map[string]string{"ubuntu": "*", "debian": "*"},
			},
			args: args{
				size:  s1,
				image: i1,
			},
			want: true,
		},
		{
			name: "default layout specific image",
			c: constraints{
				Sizes:  []string{"c1-large-x86", "c1-xlarge-x86"},
				Images: map[string]string{"ubuntu": "*", "debian": "*"},
			},
			args: args{
				size:  s1,
				image: i2,
			},
			want: true,
		},
		{
			name: "default layout specific image constraint",
			c: constraints{
				Sizes:  []string{"c1-large-x86", "c1-xlarge-x86"},
				Images: map[string]string{"ubuntu": "*", "debian": ">= 10.0.20210101"},
			},
			args: args{
				size:  s1,
				image: i2,
			},
			want: true,
		},
		{
			name: "default layout specific image constraint no match",
			c: constraints{
				Sizes:  []string{"c1-large-x86", "c1-xlarge-x86"},
				Images: map[string]string{"ubuntu": "*", "debian": ">= 10.0.20210201"},
			},
			args: args{
				size:  s1,
				image: i2,
			},
			want: false,
		},
		{
			name: "firewall layout no match",
			c: constraints{
				Sizes:  []string{"c1-large-x86", "c1-xlarge-x86"},
				Images: map[string]string{"firewall": "*"},
			},
			args: args{
				size:  s2,
				image: i1,
			},
			want: false,
		},
		{
			name: "firewall layout match",
			c: constraints{
				Sizes:  []string{"c1-large-x86", "c1-xlarge-x86"},
				Images: map[string]string{"firewall": "*"},
			},
			args: args{
				size:  s2,
				image: i3,
			},
			want: true,
		},
		{
			name: "firewall more specific layout match",
			c: constraints{
				Sizes:  []string{"c1-large-x86", "c1-xlarge-x86"},
				Images: map[string]string{"firewall": ">= 2"},
			},
			args: args{
				size:  s2,
				image: i3,
			},
			want: true,
		},
		{
			name: "firewall more specific layout no match",
			c: constraints{
				Sizes:  []string{"c1-large-x86", "c1-xlarge-x86"},
				Images: map[string]string{"firewall": ">= 3"},
			},
			args: args{
				size:  s2,
				image: i3,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			c := &FilesystemLayoutConstraints{
				Sizes:  tt.c.Sizes,
				Images: tt.c.Images,
			}
			if got := c.matches(tt.args.size, tt.args.image); got != tt.want {
				t.Errorf("FilesystemLayoutConstraint.Matches() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilesystemLayouts_From(t *testing.T) {
	type args struct {
		size  string
		image string
	}
	tests := []struct {
		name    string
		fls     FilesystemLayouts
		args    args
		want    *string
		wantErr bool
	}{
		{
			name: "simple match debian",
			fls: FilesystemLayouts{
				FilesystemLayout{
					Base: Base{ID: "default"},
					Constraints: FilesystemLayoutConstraints{
						Sizes:  []string{"c1-large-x86", "c1-xlarge-x86"},
						Images: map[string]string{"ubuntu": "*", "debian": "*"},
					},
				},
				FilesystemLayout{
					Base: Base{ID: "firewall"},
					Constraints: FilesystemLayoutConstraints{
						Sizes:  []string{"c1-large-x86", "c1-xlarge-x86"},
						Images: map[string]string{"firewall": "*"},
					},
				},
			},
			args: args{
				size:  s1,
				image: i1,
			},
			want:    strPtr("default"),
			wantErr: false,
		},
		{
			name: "simple match firewall",
			fls: FilesystemLayouts{
				FilesystemLayout{
					Base: Base{ID: "default"},
					Constraints: FilesystemLayoutConstraints{
						Sizes:  []string{"c1-large-x86", "c1-xlarge-x86"},
						Images: map[string]string{"ubuntu": "*", "debian": "*"},
					},
				},
				FilesystemLayout{
					Base: Base{ID: "firewall"},
					Constraints: FilesystemLayoutConstraints{
						Sizes:  []string{"c1-large-x86", "c1-xlarge-x86"},
						Images: map[string]string{"firewall": "*"},
					},
				},
			},
			args: args{
				size:  s1,
				image: i3,
			},
			want:    strPtr("firewall"),
			wantErr: false,
		},
		{
			name: "no match, wrong size",
			fls: FilesystemLayouts{
				FilesystemLayout{
					Base: Base{ID: "default"},
					Constraints: FilesystemLayoutConstraints{
						Sizes:  []string{"c1-large-x86", "c1-xlarge-x86"},
						Images: map[string]string{"ubuntu": "*", "debian": "*"},
					},
				},
				FilesystemLayout{
					Base: Base{ID: "firewall"},
					Constraints: FilesystemLayoutConstraints{
						Sizes:  []string{"c1-large-x86", "c1-xlarge-x86"},
						Images: map[string]string{"firewall": "*"},
					},
				},
			},
			args: args{
				size:  s3,
				image: i1,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "no match, wrong image",
			fls: FilesystemLayouts{
				FilesystemLayout{
					Base: Base{ID: "default"},
					Constraints: FilesystemLayoutConstraints{
						Sizes:  []string{"c1-large-x86", "c1-xlarge-x86"},
						Images: map[string]string{"ubuntu": "*", "debian": "*"},
					},
				},
				FilesystemLayout{
					Base: Base{ID: "firewall"},
					Constraints: FilesystemLayoutConstraints{
						Sizes:  []string{"c1-large-x86", "c1-xlarge-x86"},
						Images: map[string]string{"firewall": "*"},
					},
				},
			},
			args: args{
				size:  s1,
				image: i4,
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.fls.From(tt.args.size, tt.args.image)
			if (err != nil) != tt.wantErr {
				t.Errorf("FilesystemLayouts.From() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got == nil {
				if tt.want != nil {
					t.Errorf("FilesystemLayouts.From() got nil was not expected")
				}
				return
			}
			if !reflect.DeepEqual(got.Base.ID, *tt.want) {
				t.Errorf("FilesystemLayouts.From() = %v, want %v", got, tt.want)
			}
		})
	}
}

func strPtr(s string) *string {
	return &s
}

func TestFilesystemLayout_Matches(t *testing.T) {
	type fields struct {
		Disks []Disk
		Raid  []Raid
	}
	type args struct {
		hardware MachineHardware
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		wantErr   bool
		errString string
	}{
		{
			name: "simple match",
			fields: fields{
				Disks: []Disk{{Device: "/dev/sda"}, {Device: "/dev/sdb"}},
			},
			args:    args{hardware: MachineHardware{Disks: []BlockDevice{{Name: "/dev/sda"}, {Name: "/dev/sdb"}}}},
			wantErr: false,
		},
		{
			name: "simple match with old device naming",
			fields: fields{
				Disks: []Disk{{Device: "/dev/sda"}, {Device: "/dev/sdb"}},
			},
			args:    args{hardware: MachineHardware{Disks: []BlockDevice{{Name: "sda"}, {Name: "sdb"}}}},
			wantErr: false,
		},
		{
			name: "simple no match device missing",
			fields: fields{
				Disks: []Disk{{Device: "/dev/sda"}, {Device: "/dev/sdb"}},
			},
			args:      args{hardware: MachineHardware{Disks: []BlockDevice{{Name: "/dev/sda"}, {Name: "/dev/sdc"}}}},
			wantErr:   true,
			errString: "device:/dev/sdb does not exist on given hardware",
		},
		{
			name: "simple no match device to small",
			fields: fields{
				Disks: []Disk{
					{Device: "/dev/sda", Partitions: []DiskPartition{{Size: 100}, {Size: 100}}},
					{Device: "/dev/sdb", Partitions: []DiskPartition{{Size: 100}, {Size: 100}}}},
			},
			args: args{hardware: MachineHardware{Disks: []BlockDevice{
				{Name: "/dev/sda", Size: 300000000},
				{Name: "/dev/sdb", Size: 100000000},
			}}},
			wantErr:   true,
			errString: "device:/dev/sdb is not big enough required:200MiB, existing:95MiB",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			fl := &FilesystemLayout{
				Disks: tt.fields.Disks,
				Raid:  tt.fields.Raid,
			}
			err := fl.Matches(tt.args.hardware)
			if (err != nil) != tt.wantErr {
				t.Errorf("FilesystemLayout.Matches() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if (err != nil) && err.Error() != tt.errString {
				t.Errorf("FilesystemLayout.Matches() error = %v, errString %v", err, tt.errString)
				return
			}
		})
	}
}

func TestFilesystemLayout_Validate(t *testing.T) {
	type fields struct {
		Constraints    FilesystemLayoutConstraints
		Filesystems    []Filesystem
		Disks          []Disk
		Raid           []Raid
		VolumeGroups   []VolumeGroup
		LogicalVolumes []LogicalVolume
	}
	tests := []struct {
		name      string
		fields    fields
		wantErr   bool
		errString string
	}{
		{
			name: "valid layout",
			fields: fields{
				Constraints: FilesystemLayoutConstraints{Sizes: []string{"c1-large"}, Images: map[string]string{"ubuntu": "*"}},
				Filesystems: []Filesystem{{Path: strPtr("/boot"), Device: "/dev/sda1", Format: EXT4}, {Path: strPtr("/tmp"), Device: "tmpfs", Format: TMPFS}},
				Disks:       []Disk{{Device: "/dev/sda", Partitions: []DiskPartition{{Number: 1}}}},
			},
			wantErr: false,
		},
		{
			name: "invalid layout, wildcard image",
			fields: fields{
				Constraints: FilesystemLayoutConstraints{Sizes: []string{"c1-large"}, Images: map[string]string{"*": ""}},
				Filesystems: []Filesystem{{Path: strPtr("/boot"), Device: "/dev/sda1", Format: VFAT}},
				Disks:       []Disk{{Device: "/dev/sda", Partitions: []DiskPartition{{Number: 1}}}},
			},
			wantErr:   true,
			errString: "just '*' is not allowed as image os constraint",
		},
		{
			name: "invalid layout, wildcard size",
			fields: fields{
				Constraints: FilesystemLayoutConstraints{Sizes: []string{"c1-large*"}, Images: map[string]string{"debian": "*"}},
				Filesystems: []Filesystem{{Path: strPtr("/boot"), Device: "/dev/sda1", Format: VFAT}},
				Disks:       []Disk{{Device: "/dev/sda", Partitions: []DiskPartition{{Number: 1}}}},
			},
			wantErr:   true,
			errString: "no wildcard allowed in size constraint",
		},
		{
			name: "invalid layout, duplicate size",
			fields: fields{
				Constraints: FilesystemLayoutConstraints{Sizes: []string{"c1-large", "c1-large", "c1-xlarge"}, Images: map[string]string{"debian": "*"}},
				Filesystems: []Filesystem{{Path: strPtr("/boot"), Device: "/dev/sda1", Format: VFAT}},
				Disks:       []Disk{{Device: "/dev/sda", Partitions: []DiskPartition{{Number: 1}}}},
			},
			wantErr:   true,
			errString: "size c1-large is configured more than once",
		},
		{
			name: "invalid layout /dev/sda2 is missing",
			fields: fields{
				Filesystems: []Filesystem{{Path: strPtr("/boot"), Device: "/dev/sda1", Format: VFAT}, {Path: strPtr("/"), Device: "/dev/sda2", Format: EXT4}},
				Disks:       []Disk{{Device: "/dev/sda", Partitions: []DiskPartition{{Number: 1}}}},
			},
			wantErr:   true,
			errString: "device:/dev/sda2 for filesystem:/ is not configured",
		},
		{
			name: "invalid layout wrong Format",
			fields: fields{
				Filesystems: []Filesystem{{Path: strPtr("/boot"), Device: "/dev/sda1", Format: "xfs"}},
				Disks:       []Disk{{Device: "/dev/sda", Partitions: []DiskPartition{{Number: 1}}}},
			},
			wantErr:   true,
			errString: "filesystem:/boot format:xfs is not supported",
		},
		{
			name: "invalid layout wrong GPTType",
			fields: fields{
				Filesystems: []Filesystem{{Path: strPtr("/boot"), Device: "/dev/sda1", Format: "vfat"}},
				Disks:       []Disk{{Device: "/dev/sda", Partitions: []DiskPartition{{Number: 1, GPTType: &GPTInvalid}}}},
			},
			wantErr:   true,
			errString: "given GPTType:ff00 for partition:1 on disk:/dev/sda is not supported",
		},
		{
			name: "valid raid layout",
			fields: fields{
				Filesystems: []Filesystem{{Path: strPtr("/boot"), Device: "/dev/md1", Format: VFAT}},
				Disks: []Disk{
					{Device: "/dev/sda", Partitions: []DiskPartition{{Number: 1}}},
					{Device: "/dev/sdb", Partitions: []DiskPartition{{Number: 1}}},
				},
				Raid: []Raid{
					{ArrayName: "/dev/md1", Devices: []string{"/dev/sda1", "/dev/sdb1"}, Level: RaidLevel1},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid raid layout wrong level",
			fields: fields{
				Filesystems: []Filesystem{{Path: strPtr("/boot"), Device: "/dev/md1", Format: VFAT}},
				Disks: []Disk{
					{Device: "/dev/sda", Partitions: []DiskPartition{{Number: 1}}},
					{Device: "/dev/sdb", Partitions: []DiskPartition{{Number: 1}}},
				},
				Raid: []Raid{
					{ArrayName: "/dev/md1", Devices: []string{"/dev/sda1", "/dev/sdb1"}, Level: "6"},
				},
			},
			wantErr:   true,
			errString: "given raidlevel:6 is not supported",
		},
		{
			name: "invalid layout raid device missing",
			fields: fields{
				Filesystems: []Filesystem{{Path: strPtr("/boot"), Device: "/dev/md1"}},
				Disks: []Disk{
					{Device: "/dev/sda", Partitions: []DiskPartition{{Number: 1}}},
					{Device: "/dev/sdb", Partitions: []DiskPartition{{Number: 1}}},
				},
			},
			wantErr:   true,
			errString: "device:/dev/md1 for filesystem:/boot is not configured",
		},
		{
			name: "invalid layout device of raid missing",
			fields: fields{
				Filesystems: []Filesystem{{Path: strPtr("/boot"), Device: "/dev/md1"}},
				Disks: []Disk{
					{Device: "/dev/sda", Partitions: []DiskPartition{{Number: 1}}},
					{Device: "/dev/sdb", Partitions: []DiskPartition{{Number: 1}}},
				},
				Raid: []Raid{
					{ArrayName: "/dev/md1", Devices: []string{"/dev/sda2", "/dev/sdb2"}, Level: RaidLevel1},
				},
			},
			wantErr:   true,
			errString: "device:/dev/sda2 not provided by disk for raid:/dev/md1",
		},
		{
			name: "valid lvm layout",
			fields: fields{
				Filesystems: []Filesystem{{Path: strPtr("/boot"), Device: "/dev/vgroot/boot", Format: VFAT}},
				Disks: []Disk{
					{Device: "/dev/sda"},
					{Device: "/dev/sdb"},
				},
				VolumeGroups: []VolumeGroup{
					{Name: "vgroot", Devices: []string{"/dev/sda", "/dev/sdb"}},
				},
				LogicalVolumes: []LogicalVolume{
					{Name: "boot", VolumeGroup: "vgroot", Size: 100000000, LVMType: LVMTypeRaid1},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid lvm layout",
			fields: fields{
				Filesystems: []Filesystem{{Path: strPtr("/boot"), Device: "/dev/vg00/boot", Format: VFAT}},
				Disks: []Disk{
					{Device: "/dev/sda"},
					{Device: "/dev/sdb"},
				},
				VolumeGroups: []VolumeGroup{
					{Name: "vgroot", Devices: []string{"/dev/sda", "/dev/sdb"}},
				},
				LogicalVolumes: []LogicalVolume{
					{Name: "boot", VolumeGroup: "vgroot", Size: 100, LVMType: LVMTypeRaid1},
				},
			},
			wantErr:   true,
			errString: "device:/dev/vg00/boot for filesystem:/boot is not configured",
		},
		{
			name: "invalid lvm layout, variable size not the last one",
			fields: fields{
				Filesystems: []Filesystem{{Path: strPtr("/boot"), Device: "/dev/vgroot/boot", Format: VFAT}},
				Disks: []Disk{
					{Device: "/dev/sda"},
					{Device: "/dev/sdb"},
				},
				VolumeGroups: []VolumeGroup{
					{Name: "vgroot", Devices: []string{"/dev/sda", "/dev/sdb"}},
				},
				LogicalVolumes: []LogicalVolume{
					{Name: "boot", VolumeGroup: "vgroot", Size: 100000000, LVMType: LVMTypeRaid1},
					{Name: "/var", VolumeGroup: "vgroot", Size: 0, LVMType: LVMTypeRaid1},
					{Name: "/opt", VolumeGroup: "vgroot", Size: 20000000, LVMType: LVMTypeRaid1},
				},
			},
			wantErr:   true,
			errString: "lv:/var in vg:vgroot, variable sized lv must be the last",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			f := &FilesystemLayout{
				Constraints:    tt.fields.Constraints,
				Filesystems:    tt.fields.Filesystems,
				Disks:          tt.fields.Disks,
				Raid:           tt.fields.Raid,
				VolumeGroups:   tt.fields.VolumeGroups,
				LogicalVolumes: tt.fields.LogicalVolumes,
			}
			err := f.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("FilesystemLayout.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if (err != nil) && err.Error() != tt.errString {
				t.Errorf("FilesystemLayout.Validate()  error = %v, errString %v", err, tt.errString)
				return
			}
		})
	}
}

func TestDisk_validate(t *testing.T) {
	type fields struct {
		Device          string
		PartitionPrefix string
		Partitions      []DiskPartition
		Wipe            bool
	}
	tests := []struct {
		name      string
		fields    fields
		wantErr   bool
		errString string
	}{
		{
			name:    "simple",
			fields:  fields{Device: "/dev/sda", PartitionPrefix: "/dev/sda", Partitions: []DiskPartition{{Number: 1}}},
			wantErr: false,
		},
		{
			name: "fails because not last partition is variable",
			fields: fields{
				Device: "/dev/sda", PartitionPrefix: "/dev/sda",
				Partitions: []DiskPartition{
					{Number: 1, Size: 100},
					{Number: 2, Size: 0},
					{Number: 3, Size: 100},
				}},
			wantErr:   true,
			errString: "device:/dev/sda variable sized partition not the last one",
		},
		{
			name: "fails because not duplicate partition number",
			fields: fields{
				Device: "/dev/sda", PartitionPrefix: "/dev/sda",
				Partitions: []DiskPartition{
					{Number: 1, Size: 100},
					{Number: 2, Size: 100},
					{Number: 2, Size: 100},
				}},
			wantErr:   true,
			errString: "device:/dev/sda partition number:2 given more than once",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			d := Disk{
				Device:          tt.fields.Device,
				Partitions:      tt.fields.Partitions,
				WipeOnReinstall: tt.fields.Wipe,
			}
			err := d.validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Disk.validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if (err != nil) && err.Error() != tt.errString {
				t.Errorf("Disk.validate()  error = %v, errString %v", err, tt.errString)
				return
			}
		})
	}
}

func TestFilesystemLayouts_Validate(t *testing.T) {
	tests := []struct {
		name      string
		fls       FilesystemLayouts
		wantErr   bool
		errString string
	}{
		{
			name: "simple valid",
			fls: FilesystemLayouts{
				FilesystemLayout{Base: Base{ID: "default"}, Constraints: FilesystemLayoutConstraints{Sizes: []string{"c1-large", "c1-xlarge"}, Images: map[string]string{"ubuntu": "*", "debian": "*"}}},
				FilesystemLayout{Base: Base{ID: "firewall"}, Constraints: FilesystemLayoutConstraints{Sizes: []string{"c1-large", "c1-xlarge"}, Images: map[string]string{"firewall": "*"}}},
			},
			wantErr: false,
		},
		{
			name: "valid with open layout",
			fls: FilesystemLayouts{
				FilesystemLayout{Base: Base{ID: "default"}, Constraints: FilesystemLayoutConstraints{Sizes: []string{"c1-large", "c1-xlarge"}, Images: map[string]string{"ubuntu": "*", "debian": "*"}}},
				FilesystemLayout{Base: Base{ID: "develop-1"}, Constraints: FilesystemLayoutConstraints{Sizes: []string{}, Images: map[string]string{}}},
				FilesystemLayout{Base: Base{ID: "develop-2"}, Constraints: FilesystemLayoutConstraints{Sizes: []string{}, Images: map[string]string{}}},
				FilesystemLayout{Base: Base{ID: "firewall"}, Constraints: FilesystemLayoutConstraints{Sizes: []string{"c1-large", "c1-xlarge"}, Images: map[string]string{"firewall": "*"}}},
			},
			wantErr: false,
		},
		{
			name: "simple not overlapping, different sizes, same images",
			fls: FilesystemLayouts{
				FilesystemLayout{Base: Base{ID: "default"}, Constraints: FilesystemLayoutConstraints{Sizes: []string{"c1-large", "c1-xlarge"}, Images: map[string]string{"ubuntu": "*", "debian": "*"}}},
				FilesystemLayout{Base: Base{ID: "default2"}, Constraints: FilesystemLayoutConstraints{Sizes: []string{"s1-large", "s1-xlarge"}, Images: map[string]string{"ubuntu": "*", "debian": "*"}}},
				FilesystemLayout{Base: Base{ID: "firewall"}, Constraints: FilesystemLayoutConstraints{Sizes: []string{"c1-large", "c1-xlarge"}, Images: map[string]string{"firewall": "*"}}},
			},
			wantErr: false,
		},
		{
			name: "one overlapping, different sizes, same images",
			fls: FilesystemLayouts{
				FilesystemLayout{Base: Base{ID: "default"}, Constraints: FilesystemLayoutConstraints{Sizes: []string{"c1-large", "c1-xlarge"}, Images: map[string]string{"ubuntu": "*", "debian": ">= 10"}}},
				FilesystemLayout{Base: Base{ID: "default2"}, Constraints: FilesystemLayoutConstraints{Sizes: []string{"c1-large", "s1-large", "s1-xlarge"}, Images: map[string]string{"ubuntu": "*", "debian": "< 9"}}},
				FilesystemLayout{Base: Base{ID: "firewall"}, Constraints: FilesystemLayoutConstraints{Sizes: []string{"c1-large", "c1-xlarge"}, Images: map[string]string{"firewall": "*"}}},
			},
			wantErr:   true,
			errString: "these combinations already exist:c1-large->[ubuntu *]",
		},
		{
			name: "one overlapping, same sizes, different images",
			// FIXME fails
			fls: FilesystemLayouts{
				FilesystemLayout{Base: Base{ID: "default"}, Constraints: FilesystemLayoutConstraints{Sizes: []string{"c1-large"}, Images: map[string]string{"debian": ">= 10"}}},
				FilesystemLayout{Base: Base{ID: "default2"}, Constraints: FilesystemLayoutConstraints{Sizes: []string{"c1-large"}, Images: map[string]string{"debian": "< 10"}}},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fls.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("FilesystemLayouts.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if (err != nil) && err.Error() != tt.errString {
				t.Errorf("FilesystemLayouts.Validate()  error = %v, errString %v", err, tt.errString)
				return
			}
		})
	}
}

func Test_convertToOpAndVersion(t *testing.T) {
	tests := []struct {
		name              string
		versionconstraint string
		op                string
		version           *semver.Version
		wantErr           bool
		errString         string
	}{
		{
			name:              "simple",
			versionconstraint: ">= 10.0.1",
			op:                ">=",
			version:           semver.MustParse("10.0.1"),
			wantErr:           false,
		},
		{
			name:              "invalid no space",
			versionconstraint: ">=10.0.1",
			op:                "",
			version:           nil,
			wantErr:           true,
			errString:         "given imageconstraint:>=10.0.1 is not valid, missing space between op and version? Invalid Semantic Version",
		},
		{
			name:              "invalid version",
			versionconstraint: ">= 10.x.1",
			op:                "",
			version:           nil,
			wantErr:           true,
			errString:         "given version:10.x.1 is not valid:Invalid Semantic Version",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := convertToOpAndVersion(tt.versionconstraint)
			if (err != nil) != tt.wantErr {
				t.Errorf("convertToOpAndVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if (err != nil) && err.Error() != tt.errString {
				t.Errorf("convertToOpAndVersion  error = %v, errString %v", err, tt.errString)
				return
			}
			if got != tt.op {
				t.Errorf("convertToOpAndVersion() got = %v, want %v", got, tt.op)
			}
			if !reflect.DeepEqual(got1, tt.version) {
				t.Errorf("convertToOpAndVersion() got1 = %v, want %v", got1, tt.version)
			}
		})
	}
}

func Test_hasCollisions(t *testing.T) {
	tests := []struct {
		name               string
		versionConstraints []string
		wantErr            bool
		errString          string
	}{
		{
			name:               "simple",
			versionConstraints: []string{">= 10", "<= 9.9"},
			wantErr:            false,
		},
		{
			name:               "simple 2",
			versionConstraints: []string{">= 10", "< 10"},
			wantErr:            false,
		},
		{
			name:               "simple star match",
			versionConstraints: []string{">= 10", "<= 9.9", "*"},
			wantErr:            true,
			errString:          "at least one `*` and more than one constraint",
		},
		{
			name:               "simple versions overlap",
			versionConstraints: []string{">= 10", "<= 9.9", ">= 9.8"},
			wantErr:            true,
			errString:          "constraint:<=9.9 overlaps:>=9.8",
		},
		{
			name:               "simple versions overlap reverse",
			versionConstraints: []string{">= 9.8", "<= 9.9", ">= 10"},
			wantErr:            true,
			errString:          "constraint:>=9.8 overlaps:<=9.9",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			err := hasCollisions(tt.versionConstraints)
			if (err != nil) != tt.wantErr {
				t.Errorf("hasCollisions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if (err != nil) && err.Error() != tt.errString {
				t.Errorf("hasCollisions  error = %v, errString %v", err, tt.errString)
				return
			}
		})
	}
}

func TestToFormat(t *testing.T) {
	tests := []struct {
		name      string
		format    string
		want      Format
		wantErr   bool
		errString string
	}{
		{
			name:      "valid format",
			format:    "ext4",
			want:      EXT4,
			wantErr:   false,
			errString: "",
		},
		{
			name:      "invalid format",
			format:    "ext5",
			wantErr:   true,
			errString: "given format:ext5 is not supported, but:ext3,ext4,none,swap,tmpfs,vfat",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, err := ToFormat(tt.format)
			if (err != nil) != tt.wantErr {
				t.Errorf("ToFormat() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if (err != nil) && (err.Error() != tt.errString) {
				t.Errorf("ToFormat() error = %s, errString %s", err.Error(), tt.errString)
				return
			}
			if got != nil && *got != tt.want {
				t.Errorf("ToFormat() = %v, want %v", string(*got), tt.want)
			}
		})
	}
}

func TestToGPTType(t *testing.T) {
	tests := []struct {
		name      string
		gpttyp    string
		want      GPTType
		wantErr   bool
		errString string
	}{
		{
			name:      "valid type",
			gpttyp:    "8300",
			want:      GPTLinux,
			wantErr:   false,
			errString: "",
		},
		{
			name:      "invalid type",
			gpttyp:    "8301",
			wantErr:   true,
			errString: "given GPTType:8301 is not supported, but:8300,8e00,ef00,fd00",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, err := ToGPTType(tt.gpttyp)
			if (err != nil) != tt.wantErr {
				t.Errorf("ToGPTType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if (err != nil) && (err.Error() != tt.errString) {
				t.Errorf("ToGPTType() error = %s, errString %s", err.Error(), tt.errString)
				return
			}
			if got != nil && *got != tt.want {
				t.Errorf("ToGPTType() = %v, want %v", string(*got), tt.want)
			}
		})
	}
}

func TestToRaidLevel(t *testing.T) {
	tests := []struct {
		name      string
		level     string
		want      RaidLevel
		wantErr   bool
		errString string
	}{
		{
			name:      "valid level",
			level:     "1",
			want:      RaidLevel1,
			wantErr:   false,
			errString: "",
		},
		{
			name:      "invalid level",
			level:     "raid5",
			wantErr:   true,
			errString: "given raidlevel:raid5 is not supported, but:0,1",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, err := ToRaidLevel(tt.level)
			if (err != nil) != tt.wantErr {
				t.Errorf("ToRaidLevel() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if (err != nil) && (err.Error() != tt.errString) {
				t.Errorf("ToRaidLevel() error = %s, errString %s", err.Error(), tt.errString)
				return
			}
			if got != nil && *got != tt.want {
				t.Errorf("ToRaidLevel() = %v, want %v", string(*got), tt.want)
			}
		})
	}
}

func TestToLVMType(t *testing.T) {
	tests := []struct {
		name      string
		lvmtyp    string
		want      LVMType
		wantErr   bool
		errString string
	}{
		{
			name:      "valid lvmtype",
			lvmtyp:    "linear",
			want:      LVMTypeLinear,
			wantErr:   false,
			errString: "",
		},
		{
			name:      "invalid lvmtype",
			lvmtyp:    "raid5",
			wantErr:   true,
			errString: "given lvmtype:raid5 is not supported, but:linear,raid1,striped",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, err := ToLVMType(tt.lvmtyp)
			if (err != nil) != tt.wantErr {
				t.Errorf("ToLVMType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if (err != nil) && (err.Error() != tt.errString) {
				t.Errorf("ToLVMType() error = %s, errString %s", err.Error(), tt.errString)
				return
			}
			if got != nil && *got != tt.want {
				t.Errorf("ToLVMType() = %v, want %v", string(*got), tt.want)
			}
		})
	}
}
