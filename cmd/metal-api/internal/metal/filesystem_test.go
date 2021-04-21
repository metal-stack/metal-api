package metal

import (
	"reflect"
	"testing"
)

var (
	s1 = "c1-large-x86"
	s2 = "c1-xlarge-x86"
	s3 = "s3-large-x86"
	i1 = "debian-10"
	i3 = "firewall-2"
	i4 = "centos-7"

	GPTInvalid = GPTType("ff00")
)

func TestFilesystemLayoutConstraint_Matches(t *testing.T) {
	type constraints struct {
		Sizes  []string
		Images []string
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
				Images: []string{"ubuntu*", "debian*"},
			},
			args: args{
				size:  s1,
				image: i1,
			},
			want: true,
		},
		{
			name: "firewall layout no match",
			c: constraints{
				Sizes:  []string{"c1-large-x86", "c1-xlarge-x86"},
				Images: []string{"firewall*"},
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
				Images: []string{"firewall*"},
			},
			args: args{
				size:  s2,
				image: i3,
			},
			want: true,
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
						Images: []string{"ubuntu*", "debian*"},
					},
				},
				FilesystemLayout{
					Base: Base{ID: "firewall"},
					Constraints: FilesystemLayoutConstraints{
						Sizes:  []string{"c1-large-x86", "c1-xlarge-x86"},
						Images: []string{"firewall*"},
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
						Images: []string{"ubuntu*", "debian*"},
					},
				},
				FilesystemLayout{
					Base: Base{ID: "firewall"},
					Constraints: FilesystemLayoutConstraints{
						Sizes:  []string{"c1-large-x86", "c1-xlarge-x86"},
						Images: []string{"firewall*"},
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
						Images: []string{"ubuntu*", "debian*"},
					},
				},
				FilesystemLayout{
					Base: Base{ID: "firewall"},
					Constraints: FilesystemLayoutConstraints{
						Sizes:  []string{"c1-large-x86", "c1-xlarge-x86"},
						Images: []string{"firewall*"},
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
						Images: []string{"ubuntu*", "debian*"},
					},
				},
				FilesystemLayout{
					Base: Base{ID: "firewall"},
					Constraints: FilesystemLayoutConstraints{
						Sizes:  []string{"c1-large-x86", "c1-xlarge-x86"},
						Images: []string{"firewall*"},
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
		want      bool
		wantErr   bool
		errString string
	}{
		{
			name: "simple match",
			fields: fields{
				Disks: []Disk{{Device: "/dev/sda"}, {Device: "/dev/sdb"}},
			},
			args:    args{hardware: MachineHardware{Disks: []BlockDevice{{Name: "/dev/sda"}, {Name: "/dev/sdb"}}}},
			want:    true,
			wantErr: false,
		},
		{
			name: "simple no match device missing",
			fields: fields{
				Disks: []Disk{{Device: "/dev/sda"}, {Device: "/dev/sdb"}},
			},
			args:      args{hardware: MachineHardware{Disks: []BlockDevice{{Name: "/dev/sda"}, {Name: "/dev/sdc"}}}},
			want:      false,
			wantErr:   true,
			errString: "device:/dev/sdb does not exist on given hardware",
		},
		{
			name: "simple no match device to small",
			fields: fields{
				Disks: []Disk{
					{Device: "/dev/sda", Partitions: []DiskPartition2{{Size: 100000}, {Size: 100000}}},
					{Device: "/dev/sdb", Partitions: []DiskPartition2{{Size: 100000}, {Size: 100000}}}},
			},
			args: args{hardware: MachineHardware{Disks: []BlockDevice{
				{Name: "/dev/sda", Size: 300000},
				{Name: "/dev/sdb", Size: 100000},
			}}},
			want:      false,
			wantErr:   true,
			errString: "device:/dev/sdb is not big enough required:200000, existing:100000",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			fl := &FilesystemLayout{
				Disks: tt.fields.Disks,
				Raid:  tt.fields.Raid,
			}
			got, err := fl.Matches(tt.args.hardware)
			if (err != nil) != tt.wantErr {
				t.Errorf("FilesystemLayout.Matches() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if (err != nil) && err.Error() != tt.errString {
				t.Errorf("FilesystemLayout.Matches() error = %v, errString %v", err, tt.errString)
				return
			}
			if got != tt.want {
				t.Errorf("FilesystemLayout.Matches() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilesystemLayout_Validate(t *testing.T) {
	type fields struct {
		Constraints FilesystemLayoutConstraints
		Filesystems []Filesystem
		Disks       []Disk
		Raid        []Raid
	}
	tests := []struct {
		name      string
		fields    fields
		want      bool
		wantErr   bool
		errString string
	}{
		{
			name: "valid layout",
			fields: fields{
				Constraints: FilesystemLayoutConstraints{Sizes: []string{"c1-large"}, Images: []string{"ubuntu*"}},
				Filesystems: []Filesystem{{Path: strPtr("/boot"), Device: "/dev/sda1", Format: EXT4}, {Path: strPtr("/tmp"), Device: "tmpfs", Format: TMPFS}},
				Disks:       []Disk{{Device: "/dev/sda", PartitionPrefix: "/dev/sda", Partitions: []DiskPartition2{{Number: 1}}}},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "invalid layout, wildcard image",
			fields: fields{
				Constraints: FilesystemLayoutConstraints{Sizes: []string{"c1-large"}, Images: []string{"*"}},
				Filesystems: []Filesystem{{Path: strPtr("/boot"), Device: "/dev/sda1", Format: VFAT}},
				Disks:       []Disk{{Device: "/dev/sda", PartitionPrefix: "/dev/sda", Partitions: []DiskPartition2{{Number: 1}}}},
			},
			want:      false,
			wantErr:   true,
			errString: "just '*' is not allowed as image constraint",
		},
		{
			name: "invalid layout, wildcard size",
			fields: fields{
				Constraints: FilesystemLayoutConstraints{Sizes: []string{"c1-large*"}, Images: []string{"debian*"}},
				Filesystems: []Filesystem{{Path: strPtr("/boot"), Device: "/dev/sda1", Format: VFAT}},
				Disks:       []Disk{{Device: "/dev/sda", PartitionPrefix: "/dev/sda", Partitions: []DiskPartition2{{Number: 1}}}},
			},
			want:      false,
			wantErr:   true,
			errString: "no wildcard allowed in size constraint",
		},
		{
			name: "invalid layout /dev/sda2 is missing",
			fields: fields{
				Filesystems: []Filesystem{{Path: strPtr("/boot"), Device: "/dev/sda1", Format: VFAT}, {Path: strPtr("/"), Device: "/dev/sda2", Format: EXT4}},
				Disks:       []Disk{{Device: "/dev/sda", PartitionPrefix: "/dev/sda", Partitions: []DiskPartition2{{Number: 1}}}},
			},
			want:      false,
			wantErr:   true,
			errString: "device:/dev/sda2 for filesystem:/ is not configured as raid or device",
		},
		{
			name: "invalid layout wrong Format",
			fields: fields{
				Filesystems: []Filesystem{{Path: strPtr("/boot"), Device: "/dev/sda1", Format: "xfs"}},
				Disks:       []Disk{{Device: "/dev/sda", PartitionPrefix: "/dev/sda", Partitions: []DiskPartition2{{Number: 1}}}},
			},
			want:      false,
			wantErr:   true,
			errString: "filesystem:/boot format:xfs is not supported",
		},
		{
			name: "invalid layout wrong GPTType",
			fields: fields{
				Filesystems: []Filesystem{{Path: strPtr("/boot"), Device: "/dev/sda1", Format: "vfat"}},
				Disks:       []Disk{{Device: "/dev/sda", PartitionPrefix: "/dev/sda", Partitions: []DiskPartition2{{Number: 1, GPTType: &GPTInvalid}}}},
			},
			want:      false,
			wantErr:   true,
			errString: "given GPTType:ff00 for partition:1 on disk:/dev/sda is not supported",
		},
		{
			name: "valid raid layout",
			fields: fields{
				Filesystems: []Filesystem{{Path: strPtr("/boot"), Device: "/dev/md1", Format: VFAT}},
				Disks: []Disk{
					{Device: "/dev/sda", PartitionPrefix: "/dev/sda", Partitions: []DiskPartition2{{Number: 1}}},
					{Device: "/dev/sdb", PartitionPrefix: "/dev/sdb", Partitions: []DiskPartition2{{Number: 1}}},
				},
				Raid: []Raid{
					{Name: "/dev/md1", Devices: []string{"/dev/sda1", "/dev/sdb1"}, Level: RaidLevel1},
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "invalid raid layout wrong level",
			fields: fields{
				Filesystems: []Filesystem{{Path: strPtr("/boot"), Device: "/dev/md1", Format: VFAT}},
				Disks: []Disk{
					{Device: "/dev/sda", PartitionPrefix: "/dev/sda", Partitions: []DiskPartition2{{Number: 1}}},
					{Device: "/dev/sdb", PartitionPrefix: "/dev/sdb", Partitions: []DiskPartition2{{Number: 1}}},
				},
				Raid: []Raid{
					{Name: "/dev/md1", Devices: []string{"/dev/sda1", "/dev/sdb1"}, Level: "6"},
				},
			},
			want:      false,
			wantErr:   true,
			errString: "given raidlevel:6 is not supported",
		},
		{
			name: "invalid layout raid device missing",
			fields: fields{
				Filesystems: []Filesystem{{Path: strPtr("/boot"), Device: "/dev/md1"}},
				Disks: []Disk{
					{Device: "/dev/sda", PartitionPrefix: "/dev/sda", Partitions: []DiskPartition2{{Number: 1}}},
					{Device: "/dev/sdb", PartitionPrefix: "/dev/sdb", Partitions: []DiskPartition2{{Number: 1}}},
				},
			},
			want:      false,
			wantErr:   true,
			errString: "device:/dev/md1 for filesystem:/boot is not configured as raid or device",
		},
		{
			name: "invalid layout device of raid missing",
			fields: fields{
				Filesystems: []Filesystem{{Path: strPtr("/boot"), Device: "/dev/md1"}},
				Disks: []Disk{
					{Device: "/dev/sda", PartitionPrefix: "/dev/sda", Partitions: []DiskPartition2{{Number: 1}}},
					{Device: "/dev/sdb", PartitionPrefix: "/dev/sdb", Partitions: []DiskPartition2{{Number: 1}}},
				},
				Raid: []Raid{
					{Name: "/dev/md1", Devices: []string{"/dev/sda2", "/dev/sdb2"}, Level: RaidLevel1},
				},
			},
			want:      false,
			wantErr:   true,
			errString: "device:/dev/sda2 not provided by disk in raid:/dev/md1",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			f := &FilesystemLayout{
				Constraints: tt.fields.Constraints,
				Filesystems: tt.fields.Filesystems,
				Disks:       tt.fields.Disks,
				Raid:        tt.fields.Raid,
			}
			got, err := f.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("FilesystemLayout.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if (err != nil) && err.Error() != tt.errString {
				t.Errorf("FilesystemLayout.Validate()  error = %v, errString %v", err, tt.errString)
				return
			}
			if got != tt.want {
				t.Errorf("FilesystemLayout.Validate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDisk_validate(t *testing.T) {
	type fields struct {
		Device          string
		PartitionPrefix string
		Partitions      []DiskPartition2
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
			fields:  fields{Device: "/dev/sda", PartitionPrefix: "/dev/sda", Partitions: []DiskPartition2{{Number: 1}}},
			wantErr: false,
		},
		{
			name: "fails because not last partition is variable",
			fields: fields{
				Device: "/dev/sda", PartitionPrefix: "/dev/sda",
				Partitions: []DiskPartition2{
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
				Partitions: []DiskPartition2{
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
				PartitionPrefix: tt.fields.PartitionPrefix,
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
