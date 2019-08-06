package v1

import (
	"sort"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
)

type SwitchBase struct {
	RackID string `json:"rack_id" modelDescription:"A switch that can register at the api." description:"the id of the rack in which this switch is located"`
}

type SwitchNics []SwitchNic

type SwitchNic struct {
	MacAddress string     `json:"mac" description:"the mac address of this network interface"`
	Name       string     `json:"name" description:"the name of this network interface"`
	Vrf        string     `json:"vrf" description:"the vrf this network interface is part of" optional:"true"`
	BGPFilter  *BGPFilter `json:"filter" description:"configures the bgp filter applied at the switch port" optional:"true"`
}

type BGPFilter struct {
	CIDRs []string `json:"cidrs" description:"the cidr addresses that are allowed to be announced at this switch port"`
	VNIs  []string `json:"vnis" description:"the virtual networks that are exposed at this switch port" optional:"true"`
}

func NewBGPFilter(vnis, cidrs []string) BGPFilter {
	// Sort VNIs and CIDRs to avoid unnecessary configuration changes on leaf switches
	sort.Strings(vnis)
	sort.Strings(cidrs)
	return BGPFilter{
		VNIs:  vnis,
		CIDRs: cidrs,
	}
}

func (ss SwitchNics) ByMac() map[string]SwitchNic {
	res := make(map[string]SwitchNic)
	for i, s := range ss {
		res[s.MacAddress] = ss[i]
	}
	return res
}

type SwitchConnection struct {
	Nic       SwitchNic `json:"nic" description:"a network interface on the switch"`
	MachineID string    `json:"machine_id" optional:"true" description:"the machine id of the machine connected to the nic"`
}

type SwitchRegisterRequest struct {
	Common
	Nics        SwitchNics `json:"nics" description:"the list of network interfaces on the switch"`
	PartitionID string     `json:"partition_id" description:"the partition in which this switch is located"`
	SwitchBase
}

type SwitchResponse struct {
	Common
	SwitchBase
	Nics        SwitchNics         `json:"nics" description:"the list of network interfaces on the switch"`
	Partition   PartitionResponse  `json:"partition" description:"the partition in which this switch is located"`
	Connections []SwitchConnection `json:"connections" description:"a connection between a switch port and a machine"`
	Timestamps
}

func NewSwitchResponse(s *metal.Switch, p *metal.Partition, nics SwitchNics, cons []SwitchConnection) *SwitchResponse {
	if s == nil {
		return nil
	}

	return &SwitchResponse{
		Common: Common{
			Identifiable: Identifiable{
				ID: s.ID,
			},
			Describable: Describable{
				Name:        &s.Name,
				Description: &s.Description,
			},
		},
		SwitchBase: SwitchBase{
			RackID: s.RackID,
		},
		Nics:        nics,
		Partition:   *NewPartitionResponse(p),
		Connections: cons,
		Timestamps: Timestamps{
			Created: s.Created,
			Changed: s.Changed,
		},
	}
}

func NewSwitch(r SwitchRegisterRequest) *metal.Switch {
	nics := metal.Nics{}
	for i := range r.Nics {
		nic := metal.Nic{
			MacAddress: metal.MacAddress(r.Nics[i].MacAddress),
			Name:       r.Nics[i].Name,
			Vrf:        r.Nics[i].Vrf,
		}
		nics = append(nics, nic)
	}

	var name string
	if r.Name != nil {
		name = *r.Name
	}
	var description string
	if r.Description != nil {
		description = *r.Description
	}

	return &metal.Switch{
		Base: metal.Base{
			ID:          r.ID,
			Name:        name,
			Description: description,
		},
		PartitionID:        r.PartitionID,
		RackID:             r.RackID,
		MachineConnections: make(metal.ConnectionMap),
		Nics:               nics,
	}
}
