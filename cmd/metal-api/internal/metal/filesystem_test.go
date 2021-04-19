package metal

import (
	"reflect"
	"testing"
)

var (
	s1 = Size{
		Base: Base{ID: "c1-large-x86"},
	}
	s2 = Size{
		Base: Base{ID: "c1-xlarge-x86"},
	}
	s3 = Size{
		Base: Base{ID: "s3-large-x86"},
	}
	s4 = Size{
		Base: Base{ID: "s2-large-x86"},
	}

	i1 = Image{
		Base: Base{ID: "debian-10"},
	}
	i2 = Image{
		Base: Base{ID: "ubuntu-20.04"},
	}
	i3 = Image{
		Base: Base{ID: "firewall-2"},
	}
	i4 = Image{
		Base: Base{ID: "centos-7"},
	}
)

func TestFilesystemLayoutConstraint_Matches(t *testing.T) {
	type constraints struct {
		Sizes  []string
		Images []string
	}
	type args struct {
		size  Size
		image Image
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
		t.Run(tt.name, func(t *testing.T) {
			c := &FilesystemLayoutConstraints{
				Sizes:  tt.c.Sizes,
				Images: tt.c.Images,
			}
			if got := c.Matches(tt.args.size, tt.args.image); got != tt.want {
				t.Errorf("FilesystemLayoutConstraint.Matches() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilesystemLayouts_From(t *testing.T) {
	type args struct {
		size  Size
		image Image
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
				Filesystems: []Filesystem{{Path: strPtr("/boot"), Device: "/dev/sda1"}},
				Disks:       []Disk{{Device: "/dev/sda", PartitionPrefix: "/dev/sda", Partitions: []DiskPartition2{{Number: 1}}}},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "invalid layout /dev/sda2 is missing",
			fields: fields{
				Filesystems: []Filesystem{{Path: strPtr("/boot"), Device: "/dev/sda1"}, {Path: strPtr("/"), Device: "/dev/sda2"}},
				Disks:       []Disk{{Device: "/dev/sda", PartitionPrefix: "/dev/sda", Partitions: []DiskPartition2{{Number: 1}}}},
			},
			want:      false,
			wantErr:   true,
			errString: "device:/dev/sda2 for filesystem:/ is not configured as raid or device",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &FilesystemLayout{
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
		Device          Device
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
					{Number: 2, Size: -1},
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
		t.Run(tt.name, func(t *testing.T) {
			d := Disk{
				Device:          tt.fields.Device,
				PartitionPrefix: tt.fields.PartitionPrefix,
				Partitions:      tt.fields.Partitions,
				Wipe:            tt.fields.Wipe,
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
