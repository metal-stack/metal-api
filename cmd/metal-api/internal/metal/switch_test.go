package metal

import (
	"reflect"
	"testing"
)

func TestSwitch_ConnectMachine(t *testing.T) {
	swp1 := Nic{Name: "swp1", MacAddress: "11:11:11"}
	swp2 := Nic{Name: "swp2", MacAddress: "22:11:11"}
	someNeighbor := Nic{MacAddress: "ignored"}
	machineId := "machine-1"
	given := &Machine{
		Base: Base{ID: machineId},
		Hardware: MachineHardware{
			Nics: []Nic{
				{
					Name:      "eth0",
					Neighbors: []Nic{swp1, someNeighbor},
				},
				{
					Name:      "eth1",
					Neighbors: []Nic{swp2, someNeighbor},
				},
			},
		},
	}
	want := ConnectionMap{
		machineId: []Connection{
			{MachineID: machineId, Nic: swp1},
			{MachineID: machineId, Nic: swp2},
		},
	}

	s := Switch{
		Nics: []Nic{
			{Name: "eth0", MacAddress: "00:11:11"},
			swp1,
			swp2,
		},
		MachineConnections: ConnectionMap{},
	}

	s.ConnectMachine(given)

	if !reflect.DeepEqual(s.MachineConnections, want) {
		t.Errorf("got: %v, want:%v", s.MachineConnections, want)
	}
}
