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
			Constraint{
				MinCores:  1,
				MaxCores:  1,
				MinMemory: 1024,
				MaxMemory: 1024,
			},
		},
	}
	tinySize = Size{
		Base: Base{
			Name: "tiny",
		},
		Constraints: []Constraint{
			Constraint{
				MinCores:  1,
				MaxCores:  1,
				MinMemory: 1024,
				MaxMemory: 1077838336,
			},
		},
	}
	microAndTinySize = Size{
		Base: Base{
			Name: "microAndTiny",
		},
		Constraints: []Constraint{
			Constraint{
				MinCores:  1,
				MaxCores:  1,
				MinMemory: 1024,
				MaxMemory: 1077838336,
			},
			Constraint{
				MinCores:  1,
				MaxCores:  1,
				MinMemory: 1024,
				MaxMemory: 1024,
			},
		},
	}
)

func TestSizes_FromHardware(t *testing.T) {
	type args struct {
		hardware DeviceHardware
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
				hardware: DeviceHardware{
					CPUCores: 1,
					Memory:   1069838336,
				},
			},
			want:    &tinySize,
			wantErr: false,
		},
		{
			name: "match",
			sz:   Sizes{tinySize},
			args: args{
				hardware: DeviceHardware{
					CPUCores: 1,
					Memory:   1024,
				},
			},
			want:    &tinySize,
			wantErr: false,
		},
		{
			name: "one constraint matches",
			sz:   Sizes{microAndTinySize},
			args: args{
				hardware: DeviceHardware{
					CPUCores: 1,
					Memory:   1024,
				},
			},
			want:    &microAndTinySize,
			wantErr: false,
		},
		{
			name: "too many matches",
			sz:   Sizes{microSize, tinySize},
			args: args{
				hardware: DeviceHardware{
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
				hardware: DeviceHardware{
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
				Sz1,
				Sz999,
			},
			args: args{
				hardware: DeviceHardware{
					CPUCores: 999,
					Memory:   100,
				},
			},
			want:    &Sz999,
			wantErr: false,
		},
	}

	for _, tt := range tests {
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
	for i, f := range TestSizes {
		sizeM[f.ID] = TestSizes[i]
	}

	tests := []struct {
		name string
		sz   Sizes
		want SizeMap
	}{
		// Test Data Array (only 1 data):
		{
			name: "Test 1",
			sz:   TestSizes,
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
