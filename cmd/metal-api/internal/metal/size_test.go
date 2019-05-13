package metal

import (
	"reflect"
	"testing"
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
				Min:  1024,
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
					Memory:   1024,
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

	for _, tt := range tests {
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
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.sz.ByID(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Sizes.ByID() = %v, want %v", got, tt.want)
			}
		})
	}
}
