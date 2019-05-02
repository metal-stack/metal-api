package metal

import (
	"testing"
)

func TestMachine_HasMAC(t *testing.T) {
	type args struct {
		mac string
	}

	tests := []struct {
		name string
		d    *Machine
		args struct {
			mac string
		}
		want bool
	}{
		{
			name: "Test 1",
			d: &Machine{
				Base: Base{
					Name:        "1-core/100 B",
					Description: "a machine with 1 core(s) and 100 B of RAM",
					ID:          "5",
				},
				RackID:      "1",
				PartitionID: "1",
				Partition: Partition{
					Base: Base{
						ID:          "1",
						Name:        "partition1",
						Description: "description 1",
					},
				},
				SizeID: "1",
				Size: &Size{
					Base: Base{
						ID:          "1",
						Name:        "sz1",
						Description: "description 1",
					},
					Constraints: []Constraint{
						Constraint{
							Type: CoreConstraint,
							Min:  1,
							Max:  1,
						},
						Constraint{
							Type: MemoryConstraint,
							Min:  100,
							Max:  100,
						},
					},
				},
				Allocation: nil,
				Hardware: MachineHardware{
					Memory:   100,
					CPUCores: 1,
					Nics: Nics{
						Nic{
							MacAddress: "11:11:11:11:11:11",
						},
						Nic{
							MacAddress: "21:11:11:11:11:11",
						},
					},
					Disks: []BlockDevice{
						{
							Name: "blockdeviceName",
							Size: 1000000000000,
						},
					},
				},
			},
			args: args{
				mac: "11:11:11:11:11:11",
			},
			want: true,
		},
		{
			name: "Test 2",
			d: &Machine{
				Base:        Base{ID: "1"},
				PartitionID: "1",
				Partition: Partition{
					Base: Base{
						ID:          "1",
						Name:        "partition1",
						Description: "description 1",
					},
				},
				SizeID: "1",
				Size: &Size{
					Base: Base{
						ID:          "1",
						Name:        "sz1",
						Description: "description 1",
					},
					Constraints: []Constraint{
						Constraint{
							Type: CoreConstraint,
							Min:  1,
							Max:  1},
						Constraint{
							Type: MemoryConstraint,
							Min:  100,
							Max:  100},
					},
				},
				Allocation: &MachineAllocation{
					Name:    "d1",
					ImageID: "1",
					Project: "p1",
				},
			},
			args: args{
				mac: "11:11:11:11:11:11",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.d.HasMAC(tt.args.mac); got != tt.want {
				t.Errorf("Machine.HasMAC() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TODO: Write tests for machine allocation
