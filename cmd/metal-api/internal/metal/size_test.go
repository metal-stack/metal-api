package metal

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	mdmv1 "github.com/metal-stack/masterdata-api/api/v1"
	"github.com/metal-stack/metal-lib/pkg/pointer"
	"github.com/metal-stack/metal-lib/pkg/testcommon"
	"github.com/stretchr/testify/require"
)

var (
	mixedDiskSize = Size{
		Base: Base{
			ID: "mixedDisk",
		},
		Constraints: []Constraint{
			{
				Type: CoreConstraint,
				Min:  1,
				Max:  1,
			},
			{
				Type: MemoryConstraint,
				Min:  1024,
				Max:  1024,
			},
			{
				Type:       StorageConstraint,
				Min:        2048,
				Max:        4096,
				Identifier: "/dev/nvme*",
			},
			{
				Type:       StorageConstraint,
				Min:        0,
				Max:        1024,
				Identifier: "/dev/sd*",
			},
		},
	}
	sdaDiskSize = Size{
		Base: Base{
			ID: "sdaDisk",
		},
		Constraints: []Constraint{
			{
				Type: CoreConstraint,
				Min:  1,
				Max:  1,
			},
			{
				Type: MemoryConstraint,
				Min:  1024,
				Max:  1024,
			},
			{
				Type:       StorageConstraint,
				Min:        0,
				Max:        1024,
				Identifier: "/dev/sd*",
			},
		},
	}
	noIdentifierDiskSize = Size{
		Base: Base{
			ID: "mixedDisk",
		},
		Constraints: []Constraint{
			{
				Type: CoreConstraint,
				Min:  1,
				Max:  1,
			},
			{
				Type: MemoryConstraint,
				Min:  1024,
				Max:  1024,
			},
			{
				Type: StorageConstraint,
				Min:  0,
				Max:  1024,
			},
		},
	}
	microSize = Size{
		Base: Base{
			ID: "micro",
		},
		Constraints: []Constraint{
			{
				Type: CoreConstraint,
				Min:  1,
				Max:  1,
			},
			{
				Type: MemoryConstraint,
				Min:  1024,
				Max:  1024,
			},
			{
				Type: StorageConstraint,
				Min:  0,
				Max:  1024,
			},
		},
	}
	microOverlappingSize = Size{
		Base: Base{
			ID: "microOverlapping",
		},
		Constraints: []Constraint{
			{
				Type: CoreConstraint,
				Min:  1,
				Max:  1,
			},
			{
				Type: MemoryConstraint,
				Min:  1024,
				Max:  1024,
			},
			{
				Type: StorageConstraint,
				Min:  0,
				Max:  2048,
			},
		},
	}
	tinySize = Size{
		Base: Base{
			ID: "tiny",
		},
		Constraints: []Constraint{
			{
				Type: CoreConstraint,
				Min:  1,
				Max:  1,
			},
			{
				Type: MemoryConstraint,
				Min:  1025,
				Max:  1077838336,
			},
			{
				Type: StorageConstraint,
				Min:  1024,
				Max:  2048,
			},
		},
	}
	tinyGPUSize = Size{
		Base: Base{
			ID: "tiny gpu",
		},
		Constraints: []Constraint{
			{
				Type: CoreConstraint,
				Min:  1,
				Max:  1,
			},
			{
				Type: MemoryConstraint,
				Min:  1025,
				Max:  1077838336,
			},
			{
				Type: StorageConstraint,
				Min:  1024,
				Max:  2048,
			},
			{
				Type:       GPUConstraint,
				Max:        1,
				Min:        1,
				Identifier: "AD102GL*",
			},
		},
	}
	miniGPUSize = Size{
		Base: Base{
			ID: "mini gpu",
		},
		Constraints: []Constraint{
			{
				Type: CoreConstraint,
				Min:  1,
				Max:  1,
			},
			{
				Type: MemoryConstraint,
				Min:  1025,
				Max:  1077838336,
			},
			{
				Type: StorageConstraint,
				Min:  1024,
				Max:  2048,
			},
			{
				Type:       GPUConstraint,
				Max:        2,
				Min:        2,
				Identifier: "AD102GL*",
			},
		},
	}
	maxGPUSize = Size{
		Base: Base{
			ID: "max gpu",
		},
		Constraints: []Constraint{
			{
				Type: CoreConstraint,
				Min:  1,
				Max:  1,
			},
			{
				Type: MemoryConstraint,
				Min:  1025,
				Max:  1077838336,
			},
			{
				Type: StorageConstraint,
				Min:  1024,
				Max:  2048,
			},
			{
				Type:       GPUConstraint,
				Max:        4,
				Min:        4,
				Identifier: "H100*",
			},
		},
	}
	intelCPUSize = Size{
		Base: Base{
			ID: "intel cpu",
		},
		Constraints: []Constraint{
			{
				Type:       CoreConstraint,
				Identifier: "Intel Xeon Silver*",
				Min:        1,
				Max:        1,
			},
		},
	}
	amdCPUSize = Size{
		Base: Base{
			ID: "amd cpu",
		},
		Constraints: []Constraint{
			{
				Type:       CoreConstraint,
				Identifier: "AMD Ryzen*",
				Min:        1,
				Max:        1,
			},
		},
	}
	// Sizes
	sz1 = Size{
		Base: Base{
			ID:          "1",
			Name:        "sz1",
			Description: "description 1",
		},
		Constraints: []Constraint{
			{
				Type: CoreConstraint,
				Min:  1,
				Max:  1,
			},
			{
				Type: MemoryConstraint,
				Min:  100,
				Max:  100,
			},
		},
	}
	sz2 = Size{
		Base: Base{
			ID:          "2",
			Name:        "sz2",
			Description: "description 2",
		},
	}
	sz3 = Size{
		Base: Base{
			ID:          "3",
			Name:        "sz3",
			Description: "description 3",
		},
	}
	sz999 = Size{
		Base: Base{
			ID:          "999",
			Name:        "sz1",
			Description: "description 1",
		},
		Constraints: []Constraint{
			{
				Type: CoreConstraint,
				Min:  888,
				Max:  1111,
			},
			{
				Type: MemoryConstraint,
				Min:  100,
				Max:  100,
			},
		},
	}
	// All Sizes
	testSizes = []Size{
		sz1, sz2, sz3,
	}
)

func TestSizes_FromHardware(t *testing.T) {
	type args struct {
		hardware MachineHardware
	}
	tests := []struct {
		name    string
		sz      Sizes
		args    args
		want    *Size
		wantErr bool
	}{
		// Test Data Array:
		{
			name: "real live data",
			sz: Sizes{
				microSize,
				tinySize,
			},
			args: args{
				hardware: MachineHardware{
					MetalCPUs: []MetalCPU{
						{
							Model:   "Intel Xeon Silver",
							Cores:   1,
							Threads: 1,
						},
					},
					Memory: 1069838336,
					Disks: []BlockDevice{
						{
							Size: 1025,
						},
					},
				},
			},
			want:    &tinySize,
			wantErr: false,
		},
		{
			name: "match",
			sz:   Sizes{tinySize},
			args: args{
				hardware: MachineHardware{
					MetalCPUs: []MetalCPU{
						{
							Model:   "Intel Xeon Silver",
							Cores:   1,
							Threads: 1,
						},
					},
					Memory: 2048,
					Disks: []BlockDevice{
						{
							Size: 1025,
						},
					},
				},
			},
			want:    &tinySize,
			wantErr: false,
		},
		{
			name: "too many matches",
			sz:   Sizes{microSize, microOverlappingSize},
			args: args{
				hardware: MachineHardware{
					MetalCPUs: []MetalCPU{
						{
							Model:   "Intel Xeon Silver",
							Cores:   1,
							Threads: 1,
						},
					},
					Memory: 1024,
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "no match",
			sz:   Sizes{microSize},
			args: args{
				hardware: MachineHardware{
					MetalCPUs: []MetalCPU{
						{
							Model:   "Intel Xeon Silver",
							Cores:   1,
							Threads: 1,
						},
					},
					Memory: 2500,
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "real live data",
			sz: Sizes{
				sz1,
				sz999,
			},
			args: args{
				hardware: MachineHardware{
					MetalCPUs: []MetalCPU{
						{
							Model:   "Intel Xeon Silver",
							Cores:   999,
							Threads: 1,
						},
					},
					Memory: 100,
				},
			},
			want:    &sz999,
			wantErr: false,
		},
		{
			name: "real gpu data",
			sz: Sizes{
				sz1,
				sz999,
				tinyGPUSize,
			},
			args: args{
				hardware: MachineHardware{
					MetalCPUs: []MetalCPU{
						{
							Model:   "Intel Xeon Silver",
							Cores:   1,
							Threads: 1,
						},
					},
					Memory: 1026,
					Disks: []BlockDevice{
						{
							Size: 1026,
						},
					},
					MetalGPUs: []MetalGPU{
						{
							Vendor: "NVIDIA Corporation",
							Model:  "AD102GL [RTX 6000 Ada Generation]",
						},
					},
				},
			},
			want:    &tinyGPUSize,
			wantErr: false,
		},
		{
			name: "real larger gpu data",
			sz: Sizes{
				sz1,
				sz999,
				tinyGPUSize,
				miniGPUSize,
			},
			args: args{
				hardware: MachineHardware{
					MetalCPUs: []MetalCPU{
						{
							Model:   "Intel Xeon Silver",
							Cores:   1,
							Threads: 1,
						},
					},
					Memory: 1026,
					Disks: []BlockDevice{
						{
							Size: 1026,
						},
					},
					MetalGPUs: []MetalGPU{
						{
							Vendor: "NVIDIA Corporation",
							Model:  "AD102GL [RTX 6000 Ada Generation]",
						},
						{
							Vendor: "NVIDIA Corporation",
							Model:  "AD102GL [RTX 6000 Ada Generation]",
						},
					},
				},
			},
			want:    &miniGPUSize,
			wantErr: false,
		},
		{
			name: "real max gpu data",
			sz: Sizes{
				sz1,
				sz999,
				tinyGPUSize,
				miniGPUSize,
				maxGPUSize,
			},
			args: args{
				hardware: MachineHardware{
					MetalCPUs: []MetalCPU{
						{
							Model:   "Intel Xeon Silver",
							Cores:   1,
							Threads: 1,
						},
					},
					Memory: 1026,
					Disks: []BlockDevice{
						{
							Size: 1026,
						},
					},
					MetalGPUs: []MetalGPU{
						{
							Vendor: "NVIDIA Corporation",
							Model:  "H100",
						},
						{
							Vendor: "NVIDIA Corporation",
							Model:  "H100",
						},
						{
							Vendor: "NVIDIA Corporation",
							Model:  "H100",
						},
						{
							Vendor: "NVIDIA Corporation",
							Model:  "H100",
						},
					},
				},
			},
			want:    &maxGPUSize,
			wantErr: false,
		},
		{
			name: "mixed storage",
			sz: Sizes{
				sz1,
				sz999,
				tinyGPUSize,
				miniGPUSize,
				maxGPUSize,
				mixedDiskSize,
				sdaDiskSize,
				noIdentifierDiskSize,
			},
			args: args{
				hardware: MachineHardware{
					MetalCPUs: []MetalCPU{
						{
							Model:   "Intel Xeon Silver",
							Cores:   1,
							Threads: 1,
						},
					},
					Memory: 1024,
					Disks: []BlockDevice{
						{
							Size: 1024,
							Name: "/dev/nvme0n1",
						},
						{
							Size: 2048,
							Name: "/dev/nvme1n1",
						},
						{
							Size: 512,
							Name: "/dev/sda",
						},
					},
				},
			},
			want:    &mixedDiskSize,
			wantErr: false,
		},
		{
			name: "intel cpu",
			sz: Sizes{
				intelCPUSize,
				amdCPUSize,
			},
			args: args{
				hardware: MachineHardware{
					MetalCPUs: []MetalCPU{
						{
							Model:   "Intel Xeon Silver 4114",
							Cores:   1,
							Threads: 1,
						},
					},
				},
			},
			want:    &intelCPUSize,
			wantErr: false,
		},
		{
			name: "amd cpu",
			sz: Sizes{
				intelCPUSize,
				amdCPUSize,
			},
			args: args{
				hardware: MachineHardware{
					MetalCPUs: []MetalCPU{
						{
							Model:   "AMD Ryzen 5 8700",
							Cores:   1,
							Threads: 1,
						},
					},
				},
			},
			want:    &amdCPUSize,
			wantErr: false,
		},
	}

	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.sz.FromHardware(tt.args.hardware)
			if (err != nil) != tt.wantErr {
				t.Errorf("Sizes.FromHardware() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Sizes.FromHardware() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSizes_ByID(t *testing.T) {
	// Create the SizeMap for the Test data
	sizeM := make(SizeMap)
	for i, f := range testSizes {
		sizeM[f.ID] = testSizes[i]
	}

	tests := []struct {
		name string
		sz   Sizes
		want SizeMap
	}{
		// Test Data Array (only 1 data):
		{
			name: "TestSizes_ByID Test 1",
			sz:   testSizes,
			want: sizeM,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.sz.ByID(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Sizes.ByID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSizes_Overlaps(t *testing.T) {
	tests := []struct {
		name  string
		sz    Size
		sizes Sizes
		want  *Size
	}{
		{
			name: "non-overlapping size",
			sz: Size{
				Base: Base{
					ID: "micro",
				},
				Constraints: []Constraint{
					{
						Type: CoreConstraint,
						Min:  1,
						Max:  1,
					},
					{
						Type: MemoryConstraint,
						Min:  1024,
						Max:  1024,
					},
					{
						Type: StorageConstraint,
						Min:  0,
						Max:  1024,
					},
				},
			},
			sizes: Sizes{
				tinySize,
				Size{
					Base: Base{
						ID: "large",
					},
					Constraints: []Constraint{
						{
							Type: CoreConstraint,
							Min:  8,
							Max:  16,
						},
						{
							Type: MemoryConstraint,
							Min:  1024,
							Max:  1077838336,
						},
						{
							Type: StorageConstraint,
							Min:  1024,
							Max:  2048,
						},
					},
				},
			},
			want: nil,
		},
		{
			name: "overlapping size",
			sz: Size{
				Base: Base{
					ID: "microOverlapping",
				},
				Constraints: []Constraint{
					{
						Type: CoreConstraint,
						Min:  1,
						Max:  1,
					},
					{
						Type: MemoryConstraint,
						Min:  1024,
						Max:  1024,
					},
					{
						Type: StorageConstraint,
						Min:  0,
						Max:  2048,
					},
				},
			},
			sizes: Sizes{
				{
					Base: Base{
						ID: "micro",
					},
					Constraints: []Constraint{
						{
							Type: CoreConstraint,
							Min:  1,
							Max:  1,
						},
						{
							Type: MemoryConstraint,
							Min:  1024,
							Max:  1024,
						},
						{
							Type: StorageConstraint,
							Min:  0,
							Max:  1024,
						},
					},
				},
				{
					Base: Base{
						ID: "tiny",
					},
					Constraints: []Constraint{
						{
							Type: CoreConstraint,
							Min:  1,
							Max:  1,
						},
						{
							Type: MemoryConstraint,
							Min:  1025,
							Max:  1077838336,
						},
						{
							Type: StorageConstraint,
							Min:  1024,
							Max:  2048,
						},
					},
				},
				Size{
					Base: Base{
						ID: "large",
					},
					Constraints: []Constraint{
						{
							Type: CoreConstraint,
							Min:  8,
							Max:  16,
						},
						{
							Type: MemoryConstraint,
							Min:  1024,
							Max:  1077838336,
						},
						{
							Type: StorageConstraint,
							Min:  1024,
							Max:  2048,
						},
					},
				},
			},
			want: &microSize,
		},
		{
			name: "add incomplete size",
			sz: Size{
				Base: Base{
					ID: "microIncomplete",
				},
				Constraints: []Constraint{
					{
						Type: CoreConstraint,
						Min:  1,
						Max:  1,
					},
				},
			},
			sizes: Sizes{
				Size{
					Base: Base{
						ID: "micro",
					},
					Constraints: []Constraint{
						{
							Type: CoreConstraint,
							Min:  1,
							Max:  1,
						},
						{
							Type: MemoryConstraint,
							Min:  1024,
							Max:  1024,
						},
						{
							Type: StorageConstraint,
							Min:  0,
							Max:  1024,
						},
					},
				},
				Size{
					Base: Base{
						ID: "tiny",
					},
					Constraints: []Constraint{
						{
							Type: CoreConstraint,
							Min:  1,
							Max:  1,
						},
						{
							Type: MemoryConstraint,
							Min:  1025,
							Max:  1077838336,
						},
						{
							Type: StorageConstraint,
							Min:  1024,
							Max:  2048,
						},
					},
				},
				Size{
					Base: Base{
						ID: "large",
					},
					Constraints: []Constraint{
						{
							Type: CoreConstraint,
							Min:  8,
							Max:  16,
						},
						{
							Type: MemoryConstraint,
							Min:  1024,
							Max:  1077838336,
						},
						{
							Type: StorageConstraint,
							Min:  1024,
							Max:  2048,
						},
					},
				},
			},
			want: &microSize,
		},

		{
			name: "two different sizes",
			sz: Size{
				Base: Base{
					ID: "two different",
				},
				Constraints: []Constraint{
					{
						Type: CoreConstraint,
						Min:  1,
						Max:  1,
					},
				},
			},
			sizes: Sizes{
				Size{
					Base: Base{
						ID: "micro",
					},
					Constraints: []Constraint{
						{
							Type: MemoryConstraint,
							Min:  1024,
							Max:  1024,
						},
					},
				},
			},
			want: nil,
		},
	}

	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			err := tt.sz.Validate(nil, nil)
			require.NoError(t, err)
			got := tt.sz.Overlaps(&tt.sizes)

			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}

func TestSize_Validate(t *testing.T) {
	tests := []struct {
		name           string
		size           Size
		wantErrMessage *string
	}{
		{
			name: "cpu min and max wrong",
			size: Size{
				Base: Base{
					ID: "broken-cpu-size",
				},
				Constraints: []Constraint{
					{
						Type: CoreConstraint,
						Min:  8,
						Max:  2,
					},
				},
			},
			wantErrMessage: pointer.Pointer("size:\"broken-cpu-size\" type:\"cores\" max:2 is smaller than min:8"),
		},
		{
			name: "memory min and max wrong",
			size: Size{
				Base: Base{
					ID: "broken-memory-size",
				},
				Constraints: []Constraint{
					{
						Type: MemoryConstraint,
						Min:  8,
						Max:  2,
					},
				},
			},
			wantErrMessage: pointer.Pointer("size:\"broken-memory-size\" type:\"memory\" max:2 is smaller than min:8"),
		},
		{
			name: "storage is working",
			size: Size{
				Base: Base{
					ID: "storage-size",
				},
				Constraints: []Constraint{
					{
						Type: StorageConstraint,
						Min:  2,
						Max:  8,
					},
				},
			},
			wantErrMessage: nil,
		},
		{
			name: "two gpu constraints are allowed",
			size: Size{
				Base: Base{
					ID: "gpu-size",
				},
				Constraints: []Constraint{
					{
						Type:       GPUConstraint,
						Min:        1,
						Max:        1,
						Identifier: "A100*",
					},
					{
						Type:       GPUConstraint,
						Min:        2,
						Max:        2,
						Identifier: "H100*",
					},
				},
			},
			wantErrMessage: nil,
		},
		{
			name: "two cpu constraints are not allowed",
			size: Size{
				Base: Base{
					ID: "cpu-size",
				},
				Constraints: []Constraint{
					{
						Type:       CoreConstraint,
						Min:        1,
						Max:        1,
						Identifier: "Intel Xeon Silver",
					},
					{
						Type:       CoreConstraint,
						Min:        2,
						Max:        2,
						Identifier: "Intel Xeon Gold",
					},
				},
			},
			wantErrMessage: pointer.Pointer("size:\"cpu-size\" type:\"cores\" min:2 max:2 has duplicate constraint type"),
		},
		{
			name: "gpu size without identifier",
			size: Size{
				Base: Base{
					ID: "invalid-gpu-size",
				},
				Constraints: []Constraint{
					{
						Type: GPUConstraint,
						Min:  2,
						Max:  8,
					},
				},
			},
			wantErrMessage: pointer.Pointer("size:\"invalid-gpu-size\" type:\"gpu\" min:2 max:8 is a gpu size but has no identifier specified"),
		},
		{
			name: "storage with invalid identifier",
			size: Size{
				Base: Base{
					ID: "invalid-storage-size",
				},
				Constraints: []Constraint{
					{
						Type:       StorageConstraint,
						Identifier: "][",
						Min:        2,
						Max:        8,
					},
				},
			},
			wantErrMessage: pointer.Pointer("size:\"invalid-storage-size\" type:\"storage\" min:2 max:8 identifier:\"][\" identifier is malformed:syntax error in pattern"),
		},
		{
			name: "memory with identifier",
			size: Size{
				Base: Base{
					ID: "invalid-memory-size",
				},
				Constraints: []Constraint{
					{
						Type:       MemoryConstraint,
						Identifier: "Kingston",
						Min:        2,
						Max:        8,
					},
				},
			},
			wantErrMessage: pointer.Pointer("size:\"invalid-memory-size\" type:\"memory\" min:2 max:8 is a memory size but has a identifier specified"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.size.Validate(nil, nil)
			if err != nil {
				require.EqualError(t, err, *tt.wantErrMessage)
			}
			if err == nil && tt.wantErrMessage != nil {
				t.Errorf("expected error not raise:%s", *tt.wantErrMessage)
			}
		})
	}
}

func TestReservations_ForPartition(t *testing.T) {
	tests := []struct {
		name        string
		rs          *Reservations
		partitionID string
		want        Reservations
	}{
		{
			name:        "nil",
			rs:          nil,
			partitionID: "a",
			want:        nil,
		},
		{
			name: "correctly filtered",
			rs: &Reservations{
				{
					PartitionIDs: []string{"a", "b"},
				},
				{
					PartitionIDs: []string{"c"},
				},
				{
					PartitionIDs: []string{"a"},
				},
			},
			partitionID: "a",
			want: Reservations{
				{
					PartitionIDs: []string{"a", "b"},
				},
				{
					PartitionIDs: []string{"a"},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.rs.ForPartition(tt.partitionID); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Reservations.ForPartition() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReservations_ForProject(t *testing.T) {
	tests := []struct {
		name      string
		rs        *Reservations
		projectID string
		want      Reservations
	}{
		{
			name:      "nil",
			rs:        nil,
			projectID: "a",
			want:      nil,
		},
		{
			name: "correctly filtered",
			rs: &Reservations{
				{
					ProjectID: "a",
				},
				{
					ProjectID: "c",
				},
				{
					ProjectID: "a",
				},
			},
			projectID: "a",
			want: Reservations{
				{
					ProjectID: "a",
				},
				{
					ProjectID: "a",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.rs.ForProject(tt.projectID); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Reservations.ForProject() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReservations_Validate(t *testing.T) {
	tests := []struct {
		name       string
		partitions PartitionMap
		projects   map[string]*mdmv1.Project
		rs         *Reservations
		wantErr    error
	}{
		{
			name:       "empty reservations",
			partitions: nil,
			projects:   nil,
			rs:         nil,
			wantErr:    nil,
		},
		{
			name: "invalid amount",
			partitions: PartitionMap{
				"a": Partition{},
				"b": Partition{},
				"c": Partition{},
			},
			projects: map[string]*mdmv1.Project{
				"1": {},
				"2": {},
				"3": {},
			},
			rs: &Reservations{
				{
					Amount:       -3,
					Description:  "test",
					ProjectID:    "3",
					PartitionIDs: []string{"b"},
				},
			},
			wantErr: fmt.Errorf("amount must be a positive integer"),
		},
		{
			name: "no partitions referenced",
			partitions: PartitionMap{
				"a": Partition{},
				"b": Partition{},
				"c": Partition{},
			},
			projects: map[string]*mdmv1.Project{
				"1": {},
				"2": {},
				"3": {},
			},
			rs: &Reservations{
				{
					Amount:      3,
					Description: "test",
					ProjectID:   "3",
				},
			},
			wantErr: fmt.Errorf("at least one partition id must be specified"),
		},
		{
			name: "partition does not exist",
			partitions: PartitionMap{
				"a": Partition{},
				"b": Partition{},
				"c": Partition{},
			},
			projects: map[string]*mdmv1.Project{
				"1": {},
				"2": {},
				"3": {},
			},
			rs: &Reservations{
				{
					Amount:       3,
					Description:  "test",
					ProjectID:    "3",
					PartitionIDs: []string{"d"},
				},
			},
			wantErr: fmt.Errorf("partition must exist before creating a size reservation"),
		},
		{
			name: "partition duplicates",
			partitions: PartitionMap{
				"a": Partition{},
				"b": Partition{},
				"c": Partition{},
			},
			projects: map[string]*mdmv1.Project{
				"1": {},
				"2": {},
				"3": {},
			},
			rs: &Reservations{
				{
					Amount:       3,
					Description:  "test",
					ProjectID:    "3",
					PartitionIDs: []string{"a", "b", "c", "b"},
				},
			},
			wantErr: fmt.Errorf("partitions must not contain duplicates"),
		},
		{
			name: "no project referenced",
			partitions: PartitionMap{
				"a": Partition{},
				"b": Partition{},
				"c": Partition{},
			},
			projects: map[string]*mdmv1.Project{
				"1": {},
				"2": {},
				"3": {},
			},
			rs: &Reservations{
				{
					Amount:       3,
					Description:  "test",
					PartitionIDs: []string{"a"},
				},
			},
			wantErr: fmt.Errorf("project id must be specified"),
		},
		{
			name: "project does not exist",
			partitions: PartitionMap{
				"a": Partition{},
				"b": Partition{},
				"c": Partition{},
			},
			projects: map[string]*mdmv1.Project{
				"1": {},
				"2": {},
				"3": {},
			},
			rs: &Reservations{
				{
					Amount:       3,
					Description:  "test",
					ProjectID:    "4",
					PartitionIDs: []string{"a"},
				},
			},
			wantErr: fmt.Errorf("project must exist before creating a size reservation"),
		},
		{
			name: "valid reservation",
			partitions: PartitionMap{
				"a": Partition{},
				"b": Partition{},
				"c": Partition{},
			},
			projects: map[string]*mdmv1.Project{
				"1": {},
				"2": {},
				"3": {},
			},
			rs: &Reservations{
				{
					Amount:       3,
					Description:  "test",
					ProjectID:    "2",
					PartitionIDs: []string{"b", "c"},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.rs.Validate(tt.partitions, tt.projects)
			if diff := cmp.Diff(tt.wantErr, err, testcommon.ErrorStringComparer()); diff != "" {
				t.Errorf("error diff (-want +got):\n%s", diff)
			}
		})
	}
}

func TestConstraint_overlaps(t *testing.T) {
	tests := []struct {
		name  string
		this  Constraint
		other Constraint
		want  bool
	}{
		{
			name: "no overlap, different types",
			this: Constraint{
				Type: CoreConstraint,
			},
			other: Constraint{
				Type: GPUConstraint,
			},
			want: false,
		},
		{
			name: "no overlap, different identifiers",
			this: Constraint{
				Type:       CoreConstraint,
				Identifier: "b",
			},
			other: Constraint{
				Type:       CoreConstraint,
				Identifier: "a",
			},
			want: false,
		},

		{
			name: "no overlap, different range",
			this: Constraint{
				Type:       CoreConstraint,
				Identifier: "a",
				Min:        0,
				Max:        2,
			},
			other: Constraint{
				Type:       CoreConstraint,
				Identifier: "a",
				Min:        3,
				Max:        4,
			},
			want: false,
		},

		{
			name: "partial overlap, over range",
			this: Constraint{
				Type:       CoreConstraint,
				Identifier: "a",
				Min:        0,
				Max:        4,
			},
			other: Constraint{
				Type:       CoreConstraint,
				Identifier: "a",
				Min:        3,
				Max:        5,
			},
			want: true,
		},

		{
			name: "partial overlap, under range",
			this: Constraint{
				Type:       CoreConstraint,
				Identifier: "a",
				Min:        2,
				Max:        4,
			},
			other: Constraint{
				Type:       CoreConstraint,
				Identifier: "a",
				Min:        1,
				Max:        3,
			},
			want: true,
		},

		{
			name: "partial overlap, in range",
			this: Constraint{
				Type:       CoreConstraint,
				Identifier: "a",
				Min:        1,
				Max:        5,
			},
			other: Constraint{
				Type:       CoreConstraint,
				Identifier: "a",
				Min:        2,
				Max:        3,
			},
			want: true,
		},
		{
			name: "different disk types",
			this: Constraint{
				Type:       StorageConstraint,
				Identifier: "/dev/sd*",
				Min:        1,
				Max:        5,
			},
			other: Constraint{
				Type:       StorageConstraint,
				Identifier: "/dev/nvme*",
				Min:        1,
				Max:        5,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if got := tt.this.overlaps(tt.other); got != tt.want {
				t.Errorf("Constraint.overlaps() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSize_overlaps(t *testing.T) {
	tests := []struct {
		name  string
		this  *Size
		other *Size
		want  bool
	}{
		{
			name: "no overlap, different types",
			this: &Size{
				Constraints: []Constraint{
					{Type: MemoryConstraint},
				},
			},
			other: &Size{
				Constraints: []Constraint{
					{Type: CoreConstraint},
				},
			},
			want: false,
		},

		{
			name: "no overlap, different identifiers",
			this: &Size{
				Constraints: []Constraint{
					{Type: MemoryConstraint, Identifier: "a"},
				},
			},
			other: &Size{
				Constraints: []Constraint{
					{Type: MemoryConstraint, Identifier: "b"},
				},
			},
			want: false,
		},

		{
			name: "no overlap, different range",
			this: &Size{
				Constraints: []Constraint{
					{Type: MemoryConstraint, Identifier: "a", Min: 0, Max: 4},
				},
			},
			other: &Size{
				Constraints: []Constraint{
					{Type: MemoryConstraint, Identifier: "a", Min: 5, Max: 8},
				},
			},
			want: false,
		},

		{
			name: "no overlap, different gpus",
			this: &Size{
				Constraints: []Constraint{
					{Type: GPUConstraint, Identifier: "a", Min: 1, Max: 1},
				},
			},
			other: &Size{
				Constraints: []Constraint{
					{Type: GPUConstraint, Identifier: "a", Min: 1, Max: 1},
					{Type: GPUConstraint, Identifier: "b", Min: 2, Max: 2},
				},
			},
			want: false,
		},

		{
			name: "overlapping size",
			this: &Size{
				Constraints: []Constraint{
					{
						Type: CoreConstraint,
						Min:  1,
						Max:  1,
					},
					{
						Type: MemoryConstraint,
						Min:  1024,
						Max:  1024,
					},
					{
						Type: StorageConstraint,
						Min:  0,
						Max:  2048,
					},
				},
			},
			other: &Size{
				Constraints: []Constraint{
					{
						Type: CoreConstraint,
						Min:  1,
						Max:  1,
					},
					{
						Type: MemoryConstraint,
						Min:  1024,
						Max:  1024,
					},
					{
						Type: StorageConstraint,
						Min:  0,
						Max:  1024,
					},
				},
			},
			want: true,
		},
		{
			name: "non overlapping size",
			this: &Size{
				Constraints: []Constraint{
					{
						Type: CoreConstraint,
						Min:  1,
						Max:  1,
					},
					{
						Type: MemoryConstraint,
						Min:  1024,
						Max:  1024,
					},
					{
						Type:       StorageConstraint,
						Identifier: "/dev/sd*",
						Min:        0,
						Max:        2048,
					},
				},
			},
			other: &Size{
				Constraints: []Constraint{
					{
						Type: CoreConstraint,
						Min:  1,
						Max:  1,
					},
					{
						Type: MemoryConstraint,
						Min:  1024,
						Max:  1024,
					},
					{
						Type:       StorageConstraint,
						Identifier: "/dev/nvme*",
						Min:        0,
						Max:        2024,
					},
				},
			},
			want: false,
		},
		{
			name: "overlap, all the same",
			this: &Size{
				Constraints: []Constraint{
					{Type: MemoryConstraint, Identifier: "a", Min: 5, Max: 8},
					{Type: GPUConstraint, Identifier: "a", Min: 1, Max: 1},
					{Type: CoreConstraint, Min: 4, Max: 4},
				},
			},
			other: &Size{
				Constraints: []Constraint{
					{Type: CoreConstraint, Min: 4, Max: 4},
					{Type: GPUConstraint, Identifier: "a", Min: 1, Max: 1},
					{Type: MemoryConstraint, Identifier: "a", Min: 5, Max: 8},
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.this.overlaps(tt.other); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Size.Overlaps() = %v, want %v", got, tt.want)
			}
		})
	}
}
