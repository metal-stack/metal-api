package v1

import (
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
)

type IPBase struct {
	ProjectID string       `json:"projectid" description:"the project this ip address belongs to"`
	NetworkID string       `json:"networkid" description:"the network this ip allocate request address belongs to"`
	Type      metal.IPType `json:"type" enum:"static|ephemeral" description:"the ip type, ephemeral leads to automatic cleanup of the ip address, static will enable re-use of the ip at a later point in time"`
	Tags      []string     `json:"tags,omitempty" description:"free tags that you associate with this ip." optional:"true"`
}

type IPIdentifiable struct {
	IPAddress      string `json:"ipaddress" modelDescription:"an ip address that can be attached to a machine" description:"the address (ipv4 or ipv6) of this ip" readonly:"true"`
	AllocationUUID string `json:"allocationuuid" description:"a unique identifier for this ip address allocation, can be used to distinguish between ip address allocation over time." readonly:"true"`
}

type IPAllocateRequest struct {
	Describable
	IPBase
	MachineID *string `json:"machineid" description:"the machine id this ip should be associated with" optional:"true"`
}

type IPUpdateRequest struct {
	IPAddress string `json:"ipaddress" modelDescription:"an ip address that can be attached to a machine" description:"the address (ipv4 or ipv6) of this ip" readonly:"true"`
	Describable
	Type metal.IPType `json:"type" enum:"static|ephemeral" description:"the ip type, ephemeral leads to automatic cleanup of the ip address, static will enable re-use of the ip at a later point in time"`
	Tags []string     `json:"tags" description:"free tags that you associate with this ip." optional:"true"`
}

type IPFindRequest struct {
	datastore.IPSearchQuery
}

type IPResponse struct {
	Describable
	IPBase
	IPIdentifiable
	Timestamps
}

func NewIPResponse(ip *metal.IP) *IPResponse {
	tags := ip.Tags
	if tags == nil {
		tags = []string{}
	}
	return &IPResponse{
		Describable: Describable{
			Name:        &ip.Name,
			Description: &ip.Description,
		},
		IPBase: IPBase{
			NetworkID: ip.NetworkID,
			ProjectID: ip.ProjectID,
			Type:      ip.Type,
			Tags:      tags,
		},
		IPIdentifiable: IPIdentifiable{
			IPAddress:      ip.IPAddress,
			AllocationUUID: ip.AllocationUUID,
		},
		Timestamps: Timestamps{
			Created: ip.Created,
			Changed: ip.Changed,
		},
	}
}
