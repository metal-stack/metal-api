package v1

import (
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
)

type PartitionMgmtService struct {
	MgmtServiceAddress *string `json:"mgmtserviceaddress" description:"the address to the management service of this partition" optional:"true"`
}

type PartitionBootConfiguration struct {
	ImageURL    *string `json:"imageurl" description:"the url to download the initrd for the boot image" optional:"true"`
	KernelURL   *string `json:"kernelurl" description:"the url to download the kernel for the boot image" optional:"true"`
	CommandLine *string `json:"commandline" description:"the cmdline to the kernel for the boot image" optional:"true"`
}

type PartitionCreateRequest struct {
	Describeable
	PartitionMgmtService
	PartitionBootConfiguration PartitionBootConfiguration `json:"bootconfig" description:"the boot configuration of this partition"`
}

type PartitionUpdateRequest struct {
	Common
	PartitionMgmtService
	PartitionBootConfiguration *PartitionBootConfiguration `json:"bootconfig" description:"the boot configuration of this partition" optional:"true"`
}

type PartitionListResponse struct {
	Common
	PartitionBootConfiguration PartitionBootConfiguration `json:"bootconfig" description:"the boot configuration of this partition"`
}

type PartitionDetailResponse struct {
	PartitionListResponse
	PartitionMgmtService
	Timestamps
}

func NewPartitionDetailResponse(p *metal.Partition) *PartitionDetailResponse {
	return &PartitionDetailResponse{
		PartitionListResponse: *NewPartitionListResponse(p),
		PartitionMgmtService: PartitionMgmtService{
			MgmtServiceAddress: &p.MgmtServiceAddress,
		},
		Timestamps: Timestamps{
			Created: p.Created,
			Changed: p.Changed,
		},
	}
}

func NewPartitionListResponse(p *metal.Partition) *PartitionListResponse {
	return &PartitionListResponse{
		Common: Common{
			Identifiable: Identifiable{
				ID: p.ID,
			},
			Describeable: Describeable{
				Name:        &p.Name,
				Description: &p.Description,
			},
		},
		PartitionBootConfiguration: PartitionBootConfiguration{
			ImageURL:    &p.BootConfiguration.ImageURL,
			KernelURL:   &p.BootConfiguration.KernelURL,
			CommandLine: &p.BootConfiguration.CommandLine,
		},
	}
}
