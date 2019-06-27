package v1

import (
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
)

type IPBase struct {
	ProjectID string `json:"projectid" description:"the project this ip address belongs to"`
	NetworkID string `json:"networkid" description:"the network this ip allocate request address belongs to"`
}

type IPIdentifiable struct {
	IPAddress string `json:"ipaddress" modelDescription:"an ip address that can be attached to a machine" description:"the address (ipv4 or ipv6) of this ip" unique:"true" readonly:"true"`
	MachineID string `json:"machineid" description:"the machine this ip address belongs to, empty if not strong coupled"`
}

type IPAllocateRequest struct {
	Describable
	IPBase
}

type IPUpdateRequest struct {
	IPIdentifiable
	Describable
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
		},
		IPIdentifiable: IPIdentifiable{
			IPAddress: ip.IPAddress,
			MachineID: ip.MachineID,
		},
		Timestamps: Timestamps{
			Created: ip.Created,
			Changed: ip.Changed,
		},
	}
}
