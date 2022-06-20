package metal

import (
	"reflect"
	"testing"
)

// Gerrit and me implemented that monster in a one shot which worked.

func TestSwitch_ConnectMachine2(t *testing.T) {
	type fields struct {
		ID                 string
		Nics               []Nic
		MachineConnections ConnectionMap
		PartitionID        string
		RackID             string
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
				RackID:      "rack1",
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
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			s := Switch{
				Base: Base{
					ID: tt.fields.ID,
				},
				RackID:             tt.fields.RackID,
				Nics:               tt.fields.Nics,
				PartitionID:        tt.fields.PartitionID,
				MachineConnections: ConnectionMap{},
			}
			s.ConnectMachine(tt.machine)
			if !reflect.DeepEqual(s.MachineConnections, tt.fields.MachineConnections) {
				t.Errorf("expected:%v, got:%v", s.MachineConnections, tt.fields.MachineConnections)
			}
		})
	}
}
