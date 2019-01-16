package metal

import (
	"testing"
)

func TestDevice_HasMAC(t *testing.T) {
	type args struct {
		mac string
	}
	//tests := getAllTestStructsForTestDevice_HasMAC()

	tests := []struct {
		name string
		d    *Device
		args struct {
			mac string
		}
		want bool
	}{
		// Test Data Array (only 1 data):
		{
			name: "TestDevice_HasMAC Test 1",
			d: &Device{
				Base: Base{
					Name:        "1-core/100 B",
					Description: "a device with 1 core(s) and 100 B of RAM",
					ID:          "5",
				},
				RackID: "1",
				SiteID: "1",
				Site: Site{
					Base: Base{
						ID:          "1",
						Name:        "site1",
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
					Constraints: []Constraint{Constraint{
						MinCores:  1,
						MaxCores:  1,
						MinMemory: 100,
						MaxMemory: 100,
					}},
				},
				Allocation: nil,
				Hardware: DeviceHardware{
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
			name: "TestDevice_HasMAC Test 1",
			d: &Device{
				Base:   Base{ID: "1"},
				SiteID: "1",
				Site: Site{
					Base: Base{
						ID:          "1",
						Name:        "site1",
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
					Constraints: []Constraint{Constraint{
						MinCores:  1,
						MaxCores:  1,
						MinMemory: 100,
						MaxMemory: 100,
					}},
				},
				Allocation: &DeviceAllocation{
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
				t.Errorf("Device.HasMAC() = %v, want %v", got, tt.want)
			}
		})
	}
}
