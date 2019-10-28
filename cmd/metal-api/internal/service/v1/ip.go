package v1

import (
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/datastore"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
)

type IPBase struct {
	ProjectID string       `json:"projectid" description:"the project this ip address belongs to"`
	NetworkID string       `json:"networkid" description:"the network this ip allocate request address belongs to"`
	Type      metal.IPType `json:"iptype" default:"static" enum:"static|ephemeral" description:"the ip type, ephemeral leads to automatic cleanup of the ip address, static will enable re-use of the ip at a later point in time"`
	Tags      []string     `json:"tags" description:"free tags that you associate with this ip."`
}

type IPIdentifiable struct {
	IPAddress string `json:"ipaddress" modelDescription:"an ip address that can be attached to a machine" description:"the address (ipv4 or ipv6) of this ip" unique:"true" readonly:"true"`
}

type IPAllocateRequest struct {
	Describable
	IPBase
	MachineID *string `json:"machineid" description:"the machine id this ip should be associated with"`
	ClusterID *string `json:"clusterid" description:"the cluster id this ip should be associated with"`
}

type IPUpdateRequest struct {
	IPIdentifiable
	Describable
	Type metal.IPType `json:"iptype" enum:"static|ephemeral" description:"the ip type, ephemeral leads to automatic cleanup of the ip address, static will enable re-use of the ip at a later point in time"`
	Tags []string     `json:"tags" description:"free tags that you associate with this ip."`
}

type IPTakeRequest struct {
	IPIdentifiable
	// the cluster id to associate the ip address with.
	ClusterID *string `json:"clusterid,omitempty"`
	// the machine id to associate the ip address with.
	MachineID *string `json:"machineid,omitempty"`
	// tags to add to the ip
	Tags []string `json:"tags,omitempty"`
}

type IPReturnRequest struct {
	IPIdentifiable
	// the cluster id to associate the ip address with.
	ClusterID *string `json:"clusterid,omitempty"`
	// the machine id to associate the ip address with.
	MachineID *string `json:"machineid,omitempty"`
	// tags to add to the ip
	Tags []string `json:"tags,omitempty"`
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
	return &IPResponse{
		Describable: Describable{
			Name:        &ip.Name,
			Description: &ip.Description,
		},
		IPBase: IPBase{
			NetworkID: ip.NetworkID,
			ProjectID: ip.ProjectID,
			Type:      ip.Type,
			Tags:      ip.Tags,
		},
		IPIdentifiable: IPIdentifiable{
			IPAddress: ip.IPAddress,
		},
		Timestamps: Timestamps{
			Created: ip.Created,
			Changed: ip.Changed,
		},
	}
}
