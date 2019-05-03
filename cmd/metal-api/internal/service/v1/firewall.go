package v1

import (
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
)

type FirewallBase struct {
	PartitionID string `json:"partitionid" description:"the partition this firewall belongs to."`
	ProjectID   string `json:"projectid" description:"the project this firewall belongs to, can be empty if globally available."`
}

type FirewallImmutable struct {
	NetworkIDs []string `json:"networks" description:"the networks of this firewall, required."`
	IPs        []string `json:"ips" description:"the additional ips of this firewall, optional."`
	HA         bool     `json:"nat" description:"if set to true, this firewall is set up in a High Available manner. optional"`
}

type FirewallCreateRequest struct {
	metal.AllocateMachine // FIXME decouple from database
	//	FirewallImmutable
}

type FirewallUpdateRequest struct {
	Common
}

type FirewallListResponse struct {
	Common
	FirewallBase
	FirewallImmutable
}

type FirewallDetailResponse struct {
	metal.Machine
}

func NewFirewallDetailResponse(firewall *metal.Machine) *FirewallDetailResponse {
	// return &FirewallDetailResponse{
	// 	FirewallListResponse: *NewFirewallListResponse(firewall),
	// 	Timestamps: Timestamps{
	// 		Created: firewall.Created,
	// 		Changed: firewall.Changed,
	// 	},
	// }
	return nil
}

func NewFirewallListResponse(firewall *metal.Machine) *FirewallListResponse {

	// var networkIDs []string
	// for _, nw := range firewall.Networks {
	// 	networkIDs = append(networkIDs, nw.ID)
	// }
	// return &FirewallListResponse{
	// 	Common: Common{
	// 		Identifiable: Identifiable{
	// 			ID: firewall.ID,
	// 		},
	// 		Describeable: Describeable{
	// 			Name:        firewall.Name,
	// 			Description: firewall.Description,
	// 		},
	// 	},
	// 	FirewallBase: FirewallBase{
	// 		PartitionID: firewall.PartitionID,
	// 		ProjectID:   firewall.ProjectID,
	// 	},
	// 	FirewallImmutable: FirewallImmutable{
	// 		NetworkIDs: networkIDs,
	// 		HA:         firewall.HA,
	// 	},
	// }
	return nil
}
