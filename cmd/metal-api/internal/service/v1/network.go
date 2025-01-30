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
	Prefixes                   []string                `json:"prefixes" modelDescription:"a network which contains prefixes from which IP addresses can be allocated" description:"the prefixes of this network"`
	DestinationPrefixes        []string                `json:"destinationprefixes" modelDescription:"prefixes that are reachable within this network" description:"the destination prefixes of this network"`
	DefaultChildPrefixLength   metal.ChildPrefixLength `json:"defaultchildprefixlength" description:"if privatesuper, this defines the bitlen of child prefixes per addressfamily if not nil" optional:"true"`
	Nat                        bool                    `json:"nat" description:"if set to true, packets leaving this ipv4 network get masqueraded behind interface ip"`
	PrivateSuper               bool                    `json:"privatesuper" description:"if set to true, this network will serve as a partition's super network for the internal machine networks,there can only be one privatesuper network per partition"`
	Underlay                   bool                    `json:"underlay" description:"if set to true, this network can be used for underlay communication"`
	Vrf                        *uint                   `json:"vrf" description:"the vrf this network is associated with" optional:"true"`
	VrfShared                  *bool                   `json:"vrfshared" description:"if set to true, given vrf can be used by multiple networks, which is sometimes useful for network partitioning (default: false)" optional:"true"`
	ParentNetworkID            *string                 `json:"parentnetworkid" description:"the id of the parent network" optional:"true"`
	AdditionalAnnouncableCIDRs []string                `json:"additionalAnnouncableCIDRs,omitempty" description:"list of cidrs which are added to the route maps per tenant private network, these are typically pod- and service cidrs, can only be set for private super networks"`
}

type NetworkConsumption struct {
	IPv4 *NetworkUsage `json:"ipv4" description:"ip and prefix consumption in the ipv4 addressfamily" optional:"true" readonly:"true"`
	IPv6 *NetworkUsage `json:"ipv6" description:"ip and prefix consumption in the ipv6 addressfamily" optional:"true" readonly:"true"`
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
	DestinationPrefixes []string                `json:"destinationprefixes" description:"the destination prefixes of this network" optional:"true"`
	Nat                 *bool                   `json:"nat" description:"if set to true, packets leaving this network get masqueraded behind interface ip" optional:"true"`
	Length              metal.ChildPrefixLength `json:"length" description:"the bitlen of the prefix to allocate, defaults to defaultchildprefixlength of super prefix"`
	ParentNetworkID     *string                 `json:"parentnetworkid" description:"the parent network from which this network should be allocated"`
	AddressFamily       *metal.AddressFamily    `json:"addressfamily" description:"the addressfamily to allocate a child network defaults. If not specified, the child network inherits the addressfamilies from the parent." enum:"IPv4|IPv6"`
}

// NetworkFindRequest is used to find a Network with different criteria.
type NetworkFindRequest struct {
	datastore.NetworkSearchQuery
}

// NetworkUpdateRequest defines the properties of a Network which can be updated.
type NetworkUpdateRequest struct {
	Common
	Prefixes                   []string                `json:"prefixes" description:"the prefixes of this network" optional:"true"`
	DestinationPrefixes        []string                `json:"destinationprefixes" description:"the destination prefixes of this network" optional:"true"`
	Labels                     map[string]string       `json:"labels" description:"free labels that you associate with this network." optional:"true"`
	Shared                     *bool                   `json:"shared" description:"marks a network as shareable." optional:"true"`
	AdditionalAnnouncableCIDRs []string                `json:"additionalAnnouncableCIDRs" description:"list of cidrs which are added to the route maps per tenant private network, these are typically pod- and service cidrs, can only be set for private super networks" optional:"true"`
	DefaultChildPrefixLength   metal.ChildPrefixLength `json:"defaultchildprefixlength" description:"if privatesuper, this defines the bitlen of child prefixes per addressfamily if not nil" optional:"true"`
}

// NetworkResponse holds all properties returned in a FindNetwork or GetNetwork request.
type NetworkResponse struct {
	Common
	NetworkBase
	NetworkImmutable
	Consumption NetworkConsumption `json:"consumption" description:"consumption of ips and prefixes in this network" readonly:"true"`
	// Deprecated: The Usage field will be removed in later releases. Use Consumption instead.
	Usage NetworkUsage `json:"usage" description:"usage of IPv4 ips and prefixes in this network" readonly:"true"`
	Timestamps
}

// NewNetworkResponse converts the metal Network in the NetworkResponse visible from the API.
func NewNetworkResponse(network *metal.Network, consumption *NetworkConsumption) *NetworkResponse {
	if network == nil {
		return nil
	}

	var (
		parentNetworkID *string
	)

	if network.ParentNetworkID != "" {
		parentNetworkID = &network.ParentNetworkID
	}
	labels := network.Labels
	if labels == nil {
		labels = make(map[string]string)
	}

	response := &NetworkResponse{
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
			Prefixes:                   network.Prefixes.String(),
			DestinationPrefixes:        network.DestinationPrefixes.String(),
			DefaultChildPrefixLength:   network.DefaultChildPrefixLength,
			Nat:                        network.Nat,
			PrivateSuper:               network.PrivateSuper,
			Underlay:                   network.Underlay,
			Vrf:                        &network.Vrf,
			ParentNetworkID:            parentNetworkID,
			AdditionalAnnouncableCIDRs: network.AdditionalAnnouncableCIDRs,
		},
		Consumption: *consumption,
		Timestamps: Timestamps{
			Created: network.Created,
			Changed: network.Changed,
		},
	}

	if consumption.IPv4 != nil {
		response.Usage = *consumption.IPv4
	}

	return response
}
