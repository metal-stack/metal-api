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
		PartitionID: "1",
		RackID:      "1",
		Nics:        testNics,
		MachineConnections: ConnectionMap{
			"1": Connections{
				Connection{
					Nic: Nic{
						MacAddress: MacAddress("11:11:11:11:11:11"),
					},
					MachineID: "1",
				},
				Connection{
					Nic: Nic{
						MacAddress: MacAddress("11:11:11:11:11:22"),
					},
					MachineID: "1",
				},
			},
		},
	}
)

// Gerrit and me implemented that monster in a one shot which worked.

func TestSwitch_ConnectMachine2(t *testing.T) {
	type fields struct {
		ID                 string
		Nics               []Nic
		MachineConnections ConnectionMap
		PartitionID        string
		RackID             string
		Created            time.Time
		Changed            time.Time
		Partition          Partition
	}
	tests := []struct {
		name    string
		fields  fields
		machine *Machine
	}{
		// Test Data Array (Only 1 Value):
		{
			name: "simple connection",
			fields: fields{
				ID: "switch-1",
				Nics: []Nic{
					{
						Name:       "eth0",
						MacAddress: "00:11:11",
					},
					{
						Name:       "swp1",
						MacAddress: "11:11:11",
					},
					{
						Name:       "swp2",
						MacAddress: "22:11:11",
					},
				},
				PartitionID: "nbg1",
				Partition: Partition{
					Base: Base{
						ID: "nbg1",
					},
				},
				RackID: "rack1",
				MachineConnections: ConnectionMap{
					"machine-1": []Connection{
						{
							Nic: Nic{
								Name:       "swp1",
								MacAddress: "11:11:11",
							},
							MachineID: "machine-1",
						},
						{
							Nic: Nic{
								Name:       "swp2",
								MacAddress: "22:11:11",
							},
							MachineID: "machine-1",
						},
					},
				},
			},
			machine: &Machine{
				Base: Base{
					ID: "machine-1",
				},
				Hardware: MachineHardware{
					Nics: []Nic{
						{
							Name: "eth0",
							Neighbors: []Nic{
								{
									MacAddress: "11:11:11",
								},
								{
									MacAddress: "11:11:12",
								},
							},
						},
						{
							Name: "eth1",
							Neighbors: []Nic{
								{
									MacAddress: "22:11:11",
								},
								{
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
			s := NewSwitch(tt.fields.ID, tt.fields.RackID, tt.fields.Nics, &tt.fields.Partition)
			s.ConnectMachine(tt.machine)
			if !reflect.DeepEqual(s.MachineConnections, tt.fields.MachineConnections) {
				t.Errorf("expected:%v, got:%v", s.MachineConnections, tt.fields.MachineConnections)
			}
		})
	}
}

func TestNewSwitch(t *testing.T) {
	getNow = func() time.Time {
		return time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC)
	}
	type args struct {
		id        string
		rackid    string
		nics      Nics
		partition *Partition
	}

	tests := []struct {
		name string
		args args
		want *Switch
	}{
		// Test Data array:
		{
			name: "TestNewSwitch Test 1",
			args: args{
				id:     "1",
				rackid: "1",
				nics:   testNics,
				partition: &Partition{
					Base: Base{
						ID:          "1",
						Name:        "partition1",
						Description: "description 1",
					},
				},
			},

			want: &Switch{
				Base: Base{
					ID:      "1",
					Name:    "1",
					Created: getNow(),
					Changed: getNow(),
				},
				PartitionID:        "1",
				RackID:             "1",
				Connections:        make([]Connection, 0),
				MachineConnections: make(ConnectionMap),
				Nics:               testNics,
				Partition: Partition{
					Base: Base{
						ID:          "1",
						Name:        "partition1",
						Description: "description 1",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewSwitch(tt.args.id, tt.args.rackid, tt.args.nics, tt.args.partition); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewSwitch() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConnections_ByNic(t *testing.T) {

	testConnections := []Connection{
		{
			Nic: Nic{
				Name:       "swp1",
				MacAddress: "11:11:11",
			},
			MachineID: "machine-1",
		},
		{
			Nic: Nic{
				Name:       "swp2",
				MacAddress: "22:11:11",
			},
			MachineID: "machine-2",
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
			name: "TestSwitch_FillSwitchConnections Test 1",
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
	switches[0] = *NewSwitch("machine-1", "rack-1", testNics, &Partition{
		Base: Base{
			ID:          "1",
			Name:        "partition1",
			Description: "description 1",
		},
	})
	switches[1] = *NewSwitch("machine-2", "rack-1", testNics, &Partition{
		Base: Base{
			ID:          "1",
			Name:        "partition1",
			Description: "description 1",
		},
	})
	switches[2] = *NewSwitch("machine-3", "rack-2", testNics, &Partition{
		Base: Base{
			ID:          "2",
			Name:        "partition2",
			Description: "description 2",
		},
	})

	tests := []struct {
		name string
		args args
	}{
		// Test Data Array (Only 1 Test):
		{
			name: "TestFillAllConnections Test 1",
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

func TestSwitch_ConnectMachine(t *testing.T) {
	type args struct {
		*Machine
	}
	tests := []struct {
		name string
		s    *Switch
		args args
	}{
		// Test-Data List / Test Cases:
		{
			name: "Test 1",
			s:    &switch1,
			args: args{
				Machine: &Machine{
					Base: Base{
						Name:        "1-core/100 B",
						Description: "a machine with 1 core(s) and 100 B of RAM",
						ID:          "5",
					},
					RackID:      "1",
					PartitionID: "1",
					SizeID:      "1",
					Allocation:  nil,
					Hardware: MachineHardware{
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
			tt.s.ConnectMachine(tt.args.Machine)
		})
	}
}
