package metal

import (
	"testing"
	"time"
)

/*
Demodaten, sollten an reale Daten angepasst werden.
(Datenüberprüfungen für die Structs nicht gegeben)
*/
func TestDevice_HasMAC(t *testing.T) {
	type args struct {
		mac string
	}

	nicArray := make([]Nic, 9)
	for i := 0; i < 9; i++ {
		nicArray[i] = Nic{
			MacAddress: "11:22:33:44:55:66",
			Name:       "swp" + string(i),
			Neighbors:  nil,
		}
	}

	tests := []struct {
		name string
		d    *Device
		args args
		want bool
	}{
		// TODO: Add test cases.
		{
			name: "Test 1",
			d: &Device{
				Base: Base{
					ID: "theBaseId",
				},
				Site: Site{
					Base: Base{
						ID: "theBaseId",
					},
				},
				SiteID: "site-1",
				RackID: "rack-1",
				Size: &Size{
					Base: Base{
						ID: "theBaseId",
					},
					Constraints: []Constraint{
						{
							MinCores:  1,
							MaxCores:  2,
							MinMemory: 2000000,
							MaxMemory: 3000000,
						},
						{
							MinCores:  2,
							MaxCores:  3,
							MinMemory: 2000000,
							MaxMemory: 3000000,
						},
					},
				},
				SizeID: "size-1",
				Hardware: DeviceHardware{
					Memory:   2000200,
					CPUCores: 2,
					Nics:     nicArray,
					Disks: []BlockDevice{
						{
							Name: "Blockdevicename",
							Size: 200200,
						},
					},
				},

				Allocation: &DeviceAllocation{
					Created: time.Date(
						2009, 11, 17, 20, 34, 58, 651387237, time.UTC),
					Name:        "DeviceName",
					Description: "Description of the Device",
					LastPing: time.Date(
						2010, 11, 17, 20, 34, 58, 651387237, time.UTC),
					Tenant:  "some Tenant",
					Project: "some Project",
					Image: &Image{
						Base: Base{
							ID: "BaseId",
						},
						URL: "example.net",
					},
					ImageID:         "TheImageID",
					Cidr:            "theCidr",
					Hostname:        "theHostname",
					SSHPubKeys:      []string{"123", "passwort", "secret"},
					ConsolePassword: "toor",
				},
			},
			args: args{
				mac: "11:22:33:44:55:66",
			},
			want: true,
		},
		{
			name: "Test 1",
			d: &Device{
				Base: Base{
					ID: "theBaseId",
				},
				Site: Site{
					Base: Base{
						ID: "theBaseId",
					},
				},
				SiteID: "TheSiteId",
				RackID: "TheRackId",
				Size: &Size{
					Base: Base{
						ID: "theBaseId",
					},
					Constraints: []Constraint{
						{
							MinCores:  1,
							MaxCores:  2,
							MinMemory: 2000000,
							MaxMemory: 3000000,
						},
						{
							MinCores:  2,
							MaxCores:  3,
							MinMemory: 2000000,
							MaxMemory: 3000000,
						},
					},
				},
				SizeID: "TheSizeId",
				Hardware: DeviceHardware{
					Memory:   2000200,
					CPUCores: 2,
					Nics:     nicArray,
					Disks: []BlockDevice{
						{
							Name: "Blockdevicename",
							Size: 200200,
						},
					},
				},

				Allocation: &DeviceAllocation{
					Created: time.Date(
						2009, 11, 17, 20, 34, 58, 651387237, time.UTC),
					Name:        "DeviceName",
					Description: "Description of the Device",
					LastPing: time.Date(
						2010, 11, 17, 20, 34, 58, 651387237, time.UTC),
					Tenant:  "some Tenant",
					Project: "some Project",
					Image: &Image{
						Base: Base{
							ID: "BaseId",
						},
						URL: "example.net",
					},
					ImageID:         "TheImageID",
					Cidr:            "theCidr",
					Hostname:        "theHostname",
					SSHPubKeys:      []string{"123", "passwort", "secret"},
					ConsolePassword: "toor",
				},
			},
			args: args{
				mac: "11:22:33:44:55:67",
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
