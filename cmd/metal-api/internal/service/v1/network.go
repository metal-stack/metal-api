package v1

import (
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
)

// NetworkBase defines properties common for all Network structs.
type NetworkBase struct {
	PartitionID *string           `json:"partitionid" description:"the partition this network belongs to" optional:"true"`
	ProjectID   *string           `json:"projectid" description:"the project id this network belongs to, can be empty if globally available" optional:"true"`
	Labels      map[string]string `json:"labels" description:"free labels that you associate with this network." optional:"true"`
	Shared      *bool             `json:"shared" description:"marks a network as shareable." optional:"true"`
}

// NetworkImmutable defines the properties which are immutable in the Network.
type NetworkImmutable struct {
	Prefixes            []string `json:"prefixes" modelDescription:"a network which contains prefixes from which IP addresses can be allocated" description:"the prefixes of this network"`
	DestinationPrefixes []string `json:"destinationprefixes" modelDescription:"prefixes that are reachable within this network" description:"the destination prefixes of this network"`
	ChildPrefixLength   *uint8   `json:"childprefixlength" description:"if privatesuper, this defines the bitlen of child prefixes if not nil" optional:"true"`
	Nat                 bool     `json:"nat" description:"if set to true, packets leaving this network get masqueraded behind interface ip"`
	PrivateSuper        bool     `json:"privatesuper" description:"if set to true, this network will serve as a partition's super network for the internal machine networks,there can only be one privatesuper network per partition"`
	Underlay            bool     `json:"underlay" description:"if set to true, this network can be used for underlay communication"`
	Vrf                 *uint    `json:"vrf" description:"the vrf this network is associated with" optional:"true"`
	VrfShared           *bool    `json:"vrfshared" description:"if set to true, given vrf can be used by multiple networks, which is sometimes useful for network partitioning (default: false)" optional:"true"`
	ParentNetworkID     *string  `json:"parentnetworkid" description:"the id of the parent network" optional:"true"`
}

// NetworkUsage reports core metrics about available and used IPs or Prefixes in a Network.
type NetworkUsage struct {
	AvailableIPs      uint64 `json:"available_ips" description:"the total available IPs" readonly:"true"`
	UsedIPs           uint64 `json:"used_ips" description:"the total used IPs" readonly:"true"`
	AvailablePrefixes uint64 `json:"available_prefixes" description:"the total available 2 bit Prefixes" readonly:"true"`
	UsedPrefixes      uint64 `json:"used_prefixes" description:"the total used Prefixes" readonly:"true"`
}

// NetworkCreateRequest is used to create a new Network.
type NetworkCreateRequest struct {
	ID *string `json:"id" description:"the unique ID of this entity, auto-generated if left empty"`
	Describable
	NetworkBase
	NetworkImmutable
}

// NetworkAllocateRequest is used to allocate a Network prefix from a given Network.
type NetworkAllocateRequest struct {
	Describable
	NetworkBase
	DestinationPrefixes []string `json:"destinationprefixes" description:"the destination prefixes of this network" optional:"true"`
	Nat                 *bool    `json:"nat" description:"if set to true, packets leaving this network get masqueraded behind interface ip" optional:"true"`
	AddressFamily       *string  `json:"address_family" description:"can be ipv4 or ipv6, defaults to ipv4" optional:"true"`
	Length              *uint8   `json:"length" description:"the bitlen of the prefix to allocate, defaults to childprefixlength of super prefix" optional:"true"`
}

// AddressFamily identifies IPv4/IPv6
type AddressFamily string

const (
	// IPv4AddressFamily identifies IPv4
	IPv4AddressFamily = AddressFamily("IPv4")
	// IPv6AddressFamily identifies IPv6
	IPv6AddressFamily = AddressFamily("IPv6")
)

// ToAddressFamily will convert a string af to a AddressFamily
func ToAddressFamily(af string) AddressFamily {
	switch af {
	case "IPv4", "ipv4":
		return IPv4AddressFamily
	case "IPv6", "ipv6":
		return IPv6AddressFamily
	}
	return IPv4AddressFamily
}

// NetworkFindRequest is used to find a Network with different criteria.
type NetworkFindRequest struct {
	datastore.NetworkSearchQuery
}

// NetworkUpdateRequest defines the properties of a Network which can be updated.
type NetworkUpdateRequest struct {
	Common
	Prefixes            []string          `json:"prefixes" description:"the prefixes of this network" optional:"true"`
	DestinationPrefixes []string          `json:"destinationprefixes" description:"the destination prefixes of this network" optional:"true"`
	Labels              map[string]string `json:"labels" description:"free labels that you associate with this network." optional:"true"`
	Shared              *bool             `json:"shared" description:"marks a network as shareable." optional:"true"`
}

// NetworkResponse holds all properties returned in a FindNetwork or GetNetwork request.
type NetworkResponse struct {
	Common
	NetworkBase
	NetworkImmutable
	Usage NetworkUsage `json:"usage" description:"usage of ips and prefixes in this network" readonly:"true"`
	Timestamps
}

// NewNetworkResponse converts the metal Network in the NetworkResponse visible from the API.
func NewNetworkResponse(network *metal.Network, usage *metal.NetworkUsage) *NetworkResponse {
	if network == nil {
		return nil
	}

	var parentNetworkID *string
	if network.ParentNetworkID != "" {
		parentNetworkID = &network.ParentNetworkID
	}
	labels := network.Labels
	if labels == nil {
		labels = make(map[string]string)
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
			Labels:      labels,
			Shared:      &network.Shared,
		},
		NetworkImmutable: NetworkImmutable{
			Prefixes:            network.Prefixes.String(),
			DestinationPrefixes: network.DestinationPrefixes.String(),
			ChildPrefixLength:   network.ChildPrefixLength,
			Nat:                 network.Nat,
			PrivateSuper:        network.PrivateSuper,
			Underlay:            network.Underlay,
			Vrf:                 &network.Vrf,
			ParentNetworkID:     parentNetworkID,
		},
		Usage: NetworkUsage{
			AvailableIPs:      usage.AvailableIPs,
			UsedIPs:           usage.UsedIPs,
			AvailablePrefixes: usage.AvailablePrefixes,
			UsedPrefixes:      usage.UsedPrefixes,
		},
		Timestamps: Timestamps{
			Created: network.Created,
			Changed: network.Changed,
		},
	}
}
