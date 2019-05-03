package v1

import (
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
)

type NetworkBase struct {
	PartitionID *string `json:"partitionid" description:"the partition this network belongs to" optional:"true"`
	ProjectID   *string `json:"projectid" description:"the project this network belongs to, can be empty if globally available" optional:"true"`
}

type NetworkImmutable struct {
	Prefixes []string `json:"prefixes" description:"the prefixes of this network"`
	Nat      bool     `json:"nat" description:"if set to true, packets leaving this network get masqueraded behind interface ip"`
	Primary  bool     `json:"primary" description:"if set to true, this network is attached to a machine/firewall"`
}

type NetworkUsage struct {
	AvailableIPs      uint64 `json:"available_ips" description:"the total available IPs" readonly:"true"`
	UsedIPs           uint64 `json:"used_ips" description:"the total used IPs" readonly:"true"`
	AvailablePrefixes uint64 `json:"available_prefixes" description:"the total available Prefixes" readonly:"true"`
	UsedPrefixes      uint64 `json:"used_prefixes" description:"the total used Prefixes" readonly:"true"`
}
type NetworkCreateRequest struct {
	Describeable
	NetworkBase
	NetworkImmutable
}

type NetworkUpdateRequest struct {
	Common
	Prefixes []string `json:"prefixes" description:"the prefixes of this network" optional:"true"`
}

type NetworkListResponse struct {
	Common
	NetworkBase
	NetworkImmutable
	Usage NetworkUsage `json:"usage" description:"usage of ips and prefixes in this network" readonly:"true"`
}

type NetworkDetailResponse struct {
	NetworkListResponse
	Timestamps
}

func NewNetworkDetailResponse(network *metal.Network, usage NetworkUsage) *NetworkDetailResponse {
	return &NetworkDetailResponse{
		NetworkListResponse: *NewNetworkListResponse(network, usage),
		Timestamps: Timestamps{
			Created: network.Created,
			Changed: network.Changed,
		},
	}
}

func NewNetworkListResponse(network *metal.Network, usage NetworkUsage) *NetworkListResponse {
	var prefixes []string
	for _, p := range network.Prefixes {
		prefixes = append(prefixes, p.String())
	}
	return &NetworkListResponse{
		Common: Common{
			Identifiable: Identifiable{
				ID: network.ID,
			},
			Describeable: Describeable{
				Name:        &network.Name,
				Description: &network.Description,
			},
		},
		NetworkBase: NetworkBase{
			PartitionID: &network.PartitionID,
			ProjectID:   &network.ProjectID,
		},
		NetworkImmutable: NetworkImmutable{
			Prefixes: prefixes,
			Nat:      network.Nat,
			Primary:  network.Primary,
		},
		Usage: usage,
	}
}
