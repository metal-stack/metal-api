package v1

import (
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
)

type IPBase struct {
	ProjectID string `json:"projectid" description:"the project this ip address belongs to"`
	NetworkID string `json:"networkid" description:"the network this ip allocate request address belongs to"`
}

type IPIdentifiable struct {
	IPAddress string `json:"ipaddress" description:"the address (ipv4 or ipv6) of this ip" readonly:"true"`
}

type IPAllocateRequest struct {
	Describeable
	IPBase
}

type IPUpdateRequest struct {
	IPIdentifiable
	Describeable
}

type IPListResponse struct {
	Describeable
	IPBase
	IPIdentifiable
}

type IPDetailResponse struct {
	IPListResponse
	Timestamps
}

func NewIPDetailResponse(ip *metal.IP) *IPDetailResponse {
	return &IPDetailResponse{
		IPListResponse: *NewIPListResponse(ip),
		Timestamps: Timestamps{
			Created: ip.Created,
			Changed: ip.Changed,
		},
	}
}

func NewIPListResponse(ip *metal.IP) *IPListResponse {
	return &IPListResponse{
		Describeable: Describeable{
			Name:        &ip.Name,
			Description: &ip.Description,
		},
		IPBase: IPBase{
			NetworkID: ip.NetworkID,
			ProjectID: ip.ProjectID,
		},
		IPIdentifiable: IPIdentifiable{
			IPAddress: ip.IPAddress,
		},
	}
}
