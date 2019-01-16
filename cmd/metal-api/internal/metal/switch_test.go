package metal

import (
	"reflect"
	"testing"
	"time"
)

var (
	testNics = Nics{
		Nic{
			MacAddress: "11:11:11:11:11:11",
		},
		Nic{
			MacAddress: "21:11:11:11:11:11",
		},
	}

	// Switches
	switch1 = Switch{
		Base: Base{
			ID: "switch1",
		},
		SiteID: "1",
		RackID: "1",
		Nics:   testNics,
		DeviceConnections: ConnectionMap{
			"1": Connections{
				Connection{
					Nic: Nic{
						MacAddress: MacAddress("11:11:11:11:11:11"),
					},
					DeviceID: "1",
				},
				Connection{
					Nic: Nic{
						MacAddress: MacAddress("11:11:11:11:11:22"),
					},
					DeviceID: "1",
				},
			},
		},
	}
)

// Gerrit and me implemented that monster in a one shot which worked.

func TestSwitch_ConnectDevice2(t *testing.T) {
	type fields struct {
		ID                string
		Nics              []Nic
		DeviceConnections ConnectionMap
		SiteID            string
		RackID            string
		Created           time.Time
		Changed           time.Time
		Site              Site
	}
	tests := []struct {
		name   string
		fields fields
		device *Device
	}{
		// Test Data Array (Only 1 Value):
		{
			name: "simple connection",
			fields: fields{
				ID: "switch-1",
				Nics: []Nic{
					Nic{
						Name:       "eth0",
						MacAddress: "00:11:11",
					},
					Nic{
						Name:       "swp1",
						MacAddress: "11:11:11",
					},
					Nic{
						Name:       "swp2",
						MacAddress: "22:11:11",
					},
				},
				SiteID: "nbg1",
				Site: Site{
					Base: Base{
						ID: "nbg1",
					},
				},
				RackID: "rack1",
				DeviceConnections: ConnectionMap{
					"device-1": []Connection{
						Connection{
							Nic: Nic{
								Name:       "swp1",
								MacAddress: "11:11:11",
							},
							DeviceID: "device-1",
						},
						Connection{
							Nic: Nic{
								Name:       "swp2",
								MacAddress: "22:11:11",
							},
							DeviceID: "device-1",
						},
					},
				},
			},
			device: &Device{
				Base: Base{
					ID: "device-1",
				},
				Hardware: DeviceHardware{
					Nics: []Nic{
						Nic{
							Name: "eth0",
							Neighbors: []Nic{
								Nic{
									MacAddress: "11:11:11",
								},
								Nic{
									MacAddress: "11:11:12",
								},
							},
						},
						Nic{
							Name: "eth1",
							Neighbors: []Nic{
								Nic{
									MacAddress: "22:11:11",
								},
								Nic{
									MacAddress: "11:11:13",
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewSwitch(tt.fields.ID, tt.fields.RackID, tt.fields.Nics, &tt.fields.Site)
			s.ConnectDevice(tt.device)
			if !reflect.DeepEqual(s.DeviceConnections, tt.fields.DeviceConnections) {
				t.Errorf("expected:%v, got:%v", s.DeviceConnections, tt.fields.DeviceConnections)
			}
		})
	}
}

func TestNewSwitch(t *testing.T) {
	type args struct {
		id     string
		rackid string
		nics   Nics
		site   *Site
	}

	tests := []struct {
		name string
		args args
		want *Switch
	}{
		// Test Data array:
		{
			name: "Test 1",
			args: args{
				id:     "1",
				rackid: "1",
				nics:   testNics,
				site: &Site{
					Base: Base{
						ID:          "1",
						Name:        "site1",
						Description: "description 1",
					},
				},
			},

			want: &Switch{
				Base: Base{
					ID:   "1",
					Name: "1",
				},
				SiteID:            "1",
				RackID:            "1",
				Connections:       make([]Connection, 0),
				DeviceConnections: make(ConnectionMap),
				Nics:              testNics,
				Site: Site{
					Base: Base{
						ID:          "1",
						Name:        "site1",
						Description: "description 1",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewSwitch(tt.args.id, tt.args.rackid, tt.args.nics, tt.args.site); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewSwitch() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConnections_ByNic(t *testing.T) {

	testConnections := []Connection{
		Connection{
			Nic: Nic{
				Name:       "swp1",
				MacAddress: "11:11:11",
			},
			DeviceID: "device-1",
		},
		Connection{
			Nic: Nic{
				Name:       "swp2",
				MacAddress: "22:11:11",
			},
			DeviceID: "device-2",
		},
	}

	// Creates the Connections Map
	connectionsMap := make(map[MacAddress]Connections)
	for _, con := range testConnections {
		cons := connectionsMap[con.Nic.MacAddress]
		cons = append(cons, con)
		connectionsMap[con.Nic.MacAddress] = cons
	}

	tests := []struct {
		name string
		c    Connections
		want map[MacAddress]Connections
	}{
		// Test data Array:
		{
			name: "Test 1",
			c:    testConnections,
			want: connectionsMap,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.c.ByNic(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Connections.ByNic() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSwitch_FillSwitchConnections(t *testing.T) {

	tests := []struct {
		name string
		s    *Switch
	}{
		// Test Data Array:
		{
			name: "Test TestSwitch_FillSwitchConnections 1",
			s:    &switch1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.s.FillSwitchConnections()
		})
	}
}

func TestFillAllConnections(t *testing.T) {

	type args struct {
		sw []Switch
	}

	// Create Nics, all have all as Neighbors
	var countOfNics = 3
	nicArray := make([]Nic, countOfNics)
	for i := 0; i < countOfNics; i++ {
		nicArray[i] = Nic{
			MacAddress: MacAddress("11:11:1" + string(i)),
			Name:       "swp" + string(i),
			Neighbors:  nil,
		}
	}

	for i := 0; i < countOfNics; i++ {
		nicArray[i].Neighbors = append(nicArray[0:i], nicArray[i+1:countOfNics]...)
	}

	// Creates the Switches for the test data
	switches := make([]Switch, 3)
	switches[0] = *NewSwitch("device-1", "rack-1", testNics, &Site{
		Base: Base{
			ID:          "1",
			Name:        "site1",
			Description: "description 1",
		},
	})
	switches[1] = *NewSwitch("device-2", "rack-1", testNics, &Site{
		Base: Base{
			ID:          "1",
			Name:        "site1",
			Description: "description 1",
		},
	})
	switches[2] = *NewSwitch("device-3", "rack-2", testNics, &Site{
		Base: Base{
			ID:          "2",
			Name:        "site2",
			Description: "description 2",
		},
	})

	tests := []struct {
		name string
		args args
	}{
		// Test Data Array (Only 1 Test):
		{
			name: "Test 1",
			args: args{
				sw: switches,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			FillAllConnections(tt.args.sw)
		})
	}
}

func TestSwitch_ConnectDevice(t *testing.T) {
	type args struct {
		device *Device
	}
	tests := []struct {
		name string
		s    *Switch
		args args
	}{
		// TODO: Add test cases.
		{
			name: "Test 1",
			s:    &switch1,
			args: args{
				device: &Device{
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
					SizeID:     "1",
					Size:       &sz1,
					Allocation: nil,
					Hardware: DeviceHardware{
						Memory:   100,
						CPUCores: 1,
						Nics:     testNics,
						Disks: []BlockDevice{
							{
								Name: "blockdeviceName",
								Size: 1000000000000,
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.s.ConnectDevice(tt.args.device)
		})
	}
}
