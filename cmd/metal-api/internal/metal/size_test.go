package metal

import (
	"reflect"
	"testing"

	"github.com/metal-stack/metal-lib/pkg/pointer"
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
			err := tt.sz.Validate()
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
			err := tt.size.Validate()
			if err != nil {
				require.EqualError(t, err, *tt.wantErrMessage)
			}
			if err == nil && tt.wantErrMessage != nil {
				t.Errorf("expected error not raise:%s", *tt.wantErrMessage)
			}
		})
	}
}
