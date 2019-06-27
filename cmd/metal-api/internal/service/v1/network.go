package v1

import (
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
)

type NetworkBase struct {
	PartitionID *string `json:"partitionid" description:"the partition this network belongs to" optional:"true"`
	ProjectID   *string `json:"projectid" description:"the project this network belongs to, can be empty if globally available" optional:"true"`
}

type NetworkImmutable struct {
	Prefixes            []string `json:"prefixes" modelDescription:"a network which contains prefixes from which IP addresses can be allocated" description:"the prefixes of this network"`
	DestinationPrefixes []string `json:"destinationprefixes" modelDescription:"prefixes that are reachable within this network" description:"the destination prefixes of this network"`
	Nat                 bool     `json:"nat" description:"if set to true, packets leaving this network get masqueraded behind interface ip"`
	Primary             bool     `json:"primary" description:"if set to true, a subnetwork of this network is attached to a machine/firewall, there can only be one primary network per partition"`
	Underlay            bool     `json:"underlay" description:"if set to true, this network can be used for underlay communication"`
	Vrf                 *uint    `json:"vrf" description:"the vrf this network is associated with" optional:"true"`
	ParentNetworkID     *string  `json:"parentnetworkid" description:"the id of the parent network"`
}

type NetworkUsage struct {
	AvailableIPs      uint64 `json:"available_ips" description:"the total available IPs" readonly:"true"`
	UsedIPs           uint64 `json:"used_ips" description:"the total used IPs" readonly:"true"`
	AvailablePrefixes uint64 `json:"available_prefixes" description:"the total available Prefixes" readonly:"true"`
	UsedPrefixes      uint64 `json:"used_prefixes" description:"the total used Prefixes" readonly:"true"`
}

type NetworkCreateRequest struct {
	ID *string `json:"id" description:"the unique ID of this entity, auto-generated if left empty" unique:"true"`
	Describable
	NetworkBase
	NetworkImmutable
}

type FindNetworksRequest struct {
	ID                  *string  `json:"id" optional:"true"`
	Name                *string  `json:"name,omitempty"  optional:"true"`
	PartitionID         *string  `json:"partitionid" optional:"true"`
	ProjectID           *string  `json:"projectid" optional:"true"`
	Prefixes            []string `json:"prefixes" optional:"true"`
	DestinationPrefixes []string `json:"destinationprefixes" optional:"true"`
	Nat                 *bool    `json:"nat" optional:"true"`
	Primary             *bool    `json:"primary" optional:"true"`
	Underlay            *bool    `json:"underlay" optional:"true"`
	Vrf                 *uint    `json:"vrf" optional:"true"`
	ParentNetworkID     *string  `json:"parentnetworkid" optional:"true"`
	TenantID            *string  `json:"tenantid" optional:"true"`
}

type NetworkUpdateRequest struct {
	Common
	Prefixes []string `json:"prefixes" description:"the prefixes of this network" optional:"true"`
}

type NetworkResponse struct {
	Common
	NetworkBase
	NetworkImmutable
	Usage NetworkUsage `json:"usage" description:"usage of ips and prefixes in this network" readonly:"true"`
	Timestamps
}

func NewNetworkResponse(network *metal.Network, usage NetworkUsage) *NetworkResponse {
	var parentNetworkID *string
	if network.ParentNetworkID != "" {
		parentNetworkID = &network.ParentNetworkID
	}

	return &NetworkResponse{
		Common: Common{
			Identifiable: Identifiable{
				ID: network.ID,
			},
			Describable: Describable{
				Name:        &network.Name,
				Description: &network.Description,
			},
		},
		NetworkBase: NetworkBase{
			PartitionID: &network.PartitionID,
			ProjectID:   &network.ProjectID,
		},
		NetworkImmutable: NetworkImmutable{
			Prefixes:            network.Prefixes.String(),
			DestinationPrefixes: network.DestinationPrefixes.String(),
			Nat:                 network.Nat,
			Primary:             network.Primary,
			Underlay:            network.Underlay,
			Vrf:                 &network.Vrf,
			ParentNetworkID:     parentNetworkID,
		},
		Usage: usage,
		Timestamps: Timestamps{
			Created: network.Created,
			Changed: network.Changed,
		},
	}
}
