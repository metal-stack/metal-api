package metal

import (
	"reflect"
	"testing"
	"time"
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
				nics:   TestNicArray,
				site:   &Site1,
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
				Nics:              TestNicArray,
				Site:              Site1,
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

	// Creates the Connections Map
	connectionsMap := make(map[MacAddress]Connections)
	for _, con := range TestConnectionsArray {
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
			c:    TestConnectionsArray,
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
			s:    &Switch1,
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
	PrepareTests()

	for i := 0; i < countOfNics; i++ {
		nicArray[i].Neighbors = append(nicArray[0:i], nicArray[i+1:countOfNics]...)
	}

	// Creates the Switches for the test data
	switches := make([]Switch, 3)
	switches[0] = *NewSwitch("device-1", "rack-1", TestNicArray, &Site1)
	switches[1] = *NewSwitch("device-2", "rack-1", TestNicArray, &Site1)
	switches[2] = *NewSwitch("device-3", "rack-2", TestNicArray, &Site2)

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
			s:    &Switch1,
			args: args{
				device: &D1,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.s.ConnectDevice(tt.args.device)
		})
	}
}
