package metal

import (
	"reflect"
	"testing"
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
		{
			name: "real live data",
			sz: Sizes{
				Size{
					Name: "micro",
					Constraints: []Constraint{
						Constraint{
							MinCores:  1,
							MaxCores:  1,
							MinMemory: 1024,
							MaxMemory: 1024,
						},
					},
				},
				Size{
					Name: "tiny",
					Constraints: []Constraint{
						Constraint{
							MinCores:  1,
							MaxCores:  1,
							MinMemory: 1024,
							MaxMemory: 1077838336,
						},
					},
				},
			},
			args: args{
				hardware: DeviceHardware{
					CPUCores: 1,
					Memory:   1069838336,
				},
			},
			want: &Size{
				Name: "tiny",
				Constraints: []Constraint{
					Constraint{
						MinCores:  1,
						MaxCores:  1,
						MinMemory: 1024,
						MaxMemory: 1077838336,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "match",
			sz: Sizes{
				Size{
					Constraints: []Constraint{
						Constraint{
							MinCores:  1,
							MaxCores:  1,
							MinMemory: 100,
							MaxMemory: 200,
						},
					},
				},
			},
			args: args{
				hardware: DeviceHardware{
					CPUCores: 1,
					Memory:   150,
				},
			},
			want: &Size{
				Constraints: []Constraint{
					Constraint{
						MinCores:  1,
						MaxCores:  1,
						MinMemory: 100,
						MaxMemory: 200,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "too many matches",
			sz: Sizes{
				Size{
					Constraints: []Constraint{
						Constraint{
							MinCores:  1,
							MaxCores:  1,
							MinMemory: 100,
							MaxMemory: 200,
						},
					},
				},
				Size{
					Constraints: []Constraint{
						Constraint{
							MinCores:  1,
							MaxCores:  1,
							MinMemory: 100,
							MaxMemory: 300,
						},
					},
				},
			},
			args: args{
				hardware: DeviceHardware{
					CPUCores: 1,
					Memory:   150,
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "no match",
			sz: Sizes{
				Size{
					Constraints: []Constraint{
						Constraint{
							MinCores:  1,
							MaxCores:  1,
							MinMemory: 100,
							MaxMemory: 200,
						},
					},
				},
			},
			args: args{
				hardware: DeviceHardware{
					CPUCores: 1,
					Memory:   250,
				},
			},
			want:    nil,
			wantErr: true,
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
