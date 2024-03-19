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
	microSize = Size{
		Base: Base{
			Name: "micro",
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
			Name: "microOverlapping",
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
			Name: "tiny",
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
			Name: "tiny gpu",
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
				Type: GPUConstraint,
				GPUs: map[string]uint8{
					"AD102GL [RTX 6000 Ada Generation]": 1,
				},
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
					CPUCores: 1,
					Memory:   1069838336,
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
					CPUCores: 1,
					Memory:   2048,
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
					CPUCores: 1,
					Memory:   1024,
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
					CPUCores: 1,
					Memory:   2500,
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
					CPUCores: 999,
					Memory:   100,
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
					CPUCores: 1,
					Memory:   1026,
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
	}

	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			got, _, err := tt.sz.FromHardware(tt.args.hardware)
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
	type args struct {
		sizes Sizes
	}

	tests := []struct {
		name string
		sz   Size
		args args
		want *Size
	}{
		// Test Data Array:
		{
			name: "non-overlapping size",
			sz: Size{
				Base: Base{
					Name: "micro",
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
			args: args{
				sizes: Sizes{
					Size{
						Base: Base{
							Name: "tiny",
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
							Name: "large",
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
			},
			want: nil,
		},
		{
			name: "overlapping size",
			sz: Size{
				Base: Base{
					Name: "microOverlapping",
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
			args: args{
				sizes: Sizes{
					Size{
						Base: Base{
							Name: "micro",
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
							Name: "tiny",
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
							Name: "large",
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
			},
			want: &microSize,
		},
		{
			name: "add incomplete size",
			sz: Size{
				Base: Base{
					Name: "microIncomplete",
				},
				Constraints: []Constraint{
					{
						Type: CoreConstraint,
						Min:  1,
						Max:  1,
					},
				},
			},
			args: args{
				sizes: Sizes{
					Size{
						Base: Base{
							Name: "micro",
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
							Name: "tiny",
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
							Name: "large",
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
			},
			want: &microSize,
		},
	}

	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			err := tt.sz.Validate(nil, nil)
			require.NoError(t, err)
			got := tt.sz.Overlaps(&tt.args.sizes)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Sizes.Overlaps() = %v, want %v", got, tt.want)
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
