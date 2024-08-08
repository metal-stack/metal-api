package v1

import (
	"sort"
	"time"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
)

// SwitchPortStatus is a type alias for a string that represents the status of a switch port.
// Valid values are defined as constants in this package.
type SwitchPortStatus string

// SwitchPortStatus defines the possible statuses for a switch port.
// UNKNOWN indicates the status is not known.
// UP indicates the port is up and operational.
// DOWN indicates the port is down and not operational.
const (
	SwitchPortStatusUnknown SwitchPortStatus = "UNKNOWN"
	SwitchPortStatusUp      SwitchPortStatus = "UP"
	SwitchPortStatusDown    SwitchPortStatus = "DOWN"
)

type SwitchBase struct {
	RackID         string    `json:"rack_id" modelDescription:"A switch that can register at the api." description:"the id of the rack in which this switch is located"`
	Mode           string    `json:"mode" description:"the mode the switch currently has" optional:"true"`
	OS             *SwitchOS `json:"os" description:"the operating system the switch currently has" optional:"true"`
	ManagementIP   string    `json:"management_ip" description:"the ip address of the management interface of the switch" optional:"true"`
	ManagementUser string    `json:"management_user" description:"the user to connect to the switch" optional:"true"`
	ConsoleCommand string    `json:"console_command" description:"command to access the console of the switch" optional:"true"`
}

type SwitchOS struct {
	// TODO: do we need a distribution e.g. edgecore vs broadcom?
	Vendor           metal.SwitchOSVendor `json:"vendor" description:"the operating system vendor the switch currently has" optional:"true"`
	Version          string               `json:"version" description:"the operating system version the switch currently has" optional:"true"`
	MetalCoreVersion string               `json:"metal_core_version" description:"the version of metal-core running" optional:"true"`
}

type SwitchNics []SwitchNic

type SwitchNic struct {
	MacAddress string           `json:"mac" description:"the mac address of this network interface"`
	Name       string           `json:"name" description:"the name of this network interface"`
	Identifier string           `json:"identifier" description:"the identifier of this network interface"`
	Vrf        string           `json:"vrf" description:"the vrf this network interface is part of" optional:"true"`
	BGPFilter  *BGPFilter       `json:"filter" description:"configures the bgp filter applied at the switch port" optional:"true"`
	Actual     SwitchPortStatus `json:"actual" description:"the current state of the nic" enum:"UP|DOWN|UNKNOWN"`
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

// SwitchFindRequest is used to find a switch with different criteria.
type SwitchFindRequest struct {
	datastore.SwitchSearchQuery
}

type SwitchUpdateRequest struct {
	Common
	SwitchBase
}

type SwitchPortToggleRequest struct {
	NicName string           `json:"nic" description:"the nic of the switch you want to change"`
	Status  SwitchPortStatus `json:"status" description:"sets the port status" enum:"UP|DOWN"`
}

// SwitchNotifyRequest represents the notification sent from the switch
// to the metal-api after a sync operation. It contains the duration of
// the sync, any error that occurred, and the updated switch port states.
type SwitchNotifyRequest struct {
	Duration   time.Duration               `json:"sync_duration" description:"the duration of the switch synchronization"`
	Error      *string                     `json:"error"`
	PortStates map[string]SwitchPortStatus `json:"port_states" description:"the current switch port states"`
}

type SwitchNotifyResponse struct {
	Common
	LastSync      *SwitchSync `json:"last_sync" description:"last successful synchronization to the switch" optional:"true"`
	LastSyncError *SwitchSync `json:"last_sync_error" description:"last synchronization to the switch that was erroneous" optional:"true"`
}

type SwitchResponse struct {
	Common
	SwitchBase
	Nics          SwitchNics         `json:"nics" description:"the list of network interfaces on the switch with the desired nic states"`
	Partition     PartitionResponse  `json:"partition" description:"the partition in which this switch is located"`
	Connections   []SwitchConnection `json:"connections" description:"a connection between a switch port and a machine with the real nic states"`
	LastSync      *SwitchSync        `json:"last_sync" description:"last successful synchronization to the switch" optional:"true"`
	LastSyncError *SwitchSync        `json:"last_sync_error" description:"last synchronization to the switch that was erroneous" optional:"true"`
	Timestamps
}

type SwitchSync struct {
	Time     time.Time     `json:"time" description:"point in time when the last switch sync happened"`
	Duration time.Duration `json:"duration" description:"the duration that lat switch sync took"`
	Error    *string       `json:"error" description:"shows the error occurred during the sync" optional:"true"`
}

func NewSwitchResponse(s *metal.Switch, ss *metal.SwitchStatus, p *metal.Partition, nics SwitchNics, cons []SwitchConnection) *SwitchResponse {
	if s == nil {
		return nil
	}

	snr := NewSwitchNotifyResponse(ss)

	var os *SwitchOS
	if s.OS != nil {
		os = &SwitchOS{
			Vendor:           s.OS.Vendor,
			Version:          s.OS.Version,
			MetalCoreVersion: s.OS.MetalCoreVersion,
		}
	}

	var partition PartitionResponse
	if partitionResp := NewPartitionResponse(p); partitionResp != nil {
		partition = *partitionResp
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
			RackID:         s.RackID,
			Mode:           string(s.Mode),
			OS:             os,
			ManagementIP:   s.ManagementIP,
			ManagementUser: s.ManagementUser,
			ConsoleCommand: s.ConsoleCommand,
		},
		Nics:          nics,
		Partition:     partition,
		Connections:   cons,
		LastSync:      snr.LastSync,
		LastSyncError: snr.LastSyncError,
		Timestamps: Timestamps{
			Created: s.Created,
			Changed: s.Changed,
		},
	}
}

func NewSwitchNotifyResponse(s *metal.SwitchStatus) *SwitchNotifyResponse {
	if s == nil {
		return &SwitchNotifyResponse{}
	}

	var lastSync *SwitchSync
	if s.LastSync != nil {
		lastSync = &SwitchSync{
			Time:     s.LastSync.Time,
			Duration: s.LastSync.Duration,
		}
	}

	var lastSyncError *SwitchSync
	if s.LastSyncError != nil {
		lastSyncError = &SwitchSync{
			Time:     s.LastSyncError.Time,
			Duration: s.LastSyncError.Duration,
			Error:    s.LastSyncError.Error,
		}
	}

	return &SwitchNotifyResponse{
		Common: Common{
			Identifiable: Identifiable{
				ID: s.ID,
			},
			Describable: Describable{
				Name:        &s.Name,
				Description: &s.Description,
			},
		},
		LastSync:      lastSync,
		LastSyncError: lastSyncError,
	}
}

func NewSwitch(r SwitchRegisterRequest) *metal.Switch {
	nics := metal.Nics{}
	for i := range r.Nics {
		nic := metal.Nic{
			MacAddress: metal.MacAddress(r.Nics[i].MacAddress),
			Name:       r.Nics[i].Name,
			Identifier: r.Nics[i].Identifier,
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

	var os *metal.SwitchOS
	if r.OS != nil {
		os = &metal.SwitchOS{
			Vendor:           r.OS.Vendor,
			Version:          r.OS.Version,
			MetalCoreVersion: r.OS.MetalCoreVersion,
		}
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
		OS:                 os,
		ManagementIP:       r.ManagementIP,
		ManagementUser:     r.ManagementUser,
		ConsoleCommand:     r.ConsoleCommand,
	}
}
