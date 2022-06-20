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

func TestConnectionMap_ByNicName(t *testing.T) {
	tests := []struct {
		name    string
		c       ConnectionMap
		want    map[string]Connection
		wantErr bool
	}{
		{
			"duplicate connections throws error",
			ConnectionMap{
				"machine-1": []Connection{{Nic: Nic{Name: "duplicate"}}},
				"machine-2": []Connection{{Nic: Nic{Name: "duplicate"}}},
			},
			nil,
			true,
		},
		{
			"happy case",
			ConnectionMap{
				"machine-1": []Connection{
					{Nic: Nic{Name: "eth0"}},
					{Nic: Nic{Name: "eth1", MacAddress: "I will be preserved"}},
				},
				"machine-2": []Connection{{Nic: Nic{Name: "eth2"}}},
			},
			map[string]Connection{
				"eth0": {Nic: Nic{Name: "eth0"}},
				"eth1": {Nic: Nic{Name: "eth1", MacAddress: "I will be preserved"}},
				"eth2": {Nic: Nic{Name: "eth2"}},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.c.ByNicName()
			if (err != nil) != tt.wantErr {
				t.Errorf("ByNicName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ByNicName() got = %v, want %v", got, tt.want)
			}
		})
	}
}
