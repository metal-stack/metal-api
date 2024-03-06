package v1

import (
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
)

type PartitionBase struct {
	MgmtServiceAddress          *string           `json:"mgmtserviceaddress" description:"the address to the management service of this partition" optional:"true"`
	PrivateNetworkPrefixLength  *int              `json:"privatenetworkprefixlength" description:"the length of private networks for the machine's child networks in this partition, default 22" optional:"true" minimum:"16" maximum:"30"`
	PartitionWaitingPoolMinSize *string           `json:"waitingpoolminsize" description:"the minimum waiting pool size of this partition" optional:"true"`
	PartitionWaitingPoolMaxSize *string           `json:"waitingpoolmaxsize" description:"the maximum waiting pool size of this partition" optional:"true"`
	Labels                      map[string]string `json:"labels" description:"free labels that you associate with this partition" optional:"true"`
}

type PartitionBootConfiguration struct {
	ImageURL    *string `json:"imageurl" modelDescription:"a partition has a distinct location in a data center, individual entities belong to a partition" description:"the url to download the initrd for the boot image" optional:"true"`
	KernelURL   *string `json:"kernelurl" description:"the url to download the kernel for the boot image" optional:"true"`
	CommandLine *string `json:"commandline" description:"the cmdline to the kernel for the boot image" optional:"true"`
}

type PartitionCreateRequest struct {
	Common
	PartitionBase
	PartitionBootConfiguration PartitionBootConfiguration `json:"bootconfig" description:"the boot configuration of this partition"`
}

type PartitionUpdateRequest struct {
	Common
	MgmtServiceAddress          *string                     `json:"mgmtserviceaddress" description:"the address to the management service of this partition" optional:"true"`
	PartitionBootConfiguration  *PartitionBootConfiguration `json:"bootconfig" description:"the boot configuration of this partition" optional:"true"`
	PartitionWaitingPoolMinSize *string                     `json:"waitingpoolminsize" description:"the minimum waiting pool size of this partition" optional:"true"`
	PartitionWaitingPoolMaxSize *string                     `json:"waitingpoolmaxsize" description:"the maximum waiting pool size of this partition" optional:"true"`
	Labels                      map[string]string           `json:"labels" description:"free labels that you associate with this partition" optional:"true"`
}

type PartitionResponse struct {
	Common
	PartitionBase
	PartitionBootConfiguration PartitionBootConfiguration `json:"bootconfig" description:"the boot configuration of this partition"`
	Timestamps
	Labels map[string]string `json:"labels" description:"free labels that you associate with this partition" optional:"true"`
}

type PartitionCapacityRequest struct {
	ID   *string `json:"id" description:"the id of the partition" optional:"true"`
	Size *string `json:"sizeid" description:"the size to filter for" optional:"true"`
}

type ServerCapacities []*ServerCapacity

type PartitionCapacity struct {
	Common
	ServerCapacities ServerCapacities `json:"servers" description:"servers available in this partition"`
}

type ServerCapacity struct {
	Size             string   `json:"size" description:"the size of the server"`
	Total            int      `json:"total" description:"total amount of servers with this size"`
	Free             int      `json:"free" description:"free servers with this size"`
	Allocated        int      `json:"allocated" description:"allocated servers with this size"`
	Reservations     int      `json:"reservations" description:"the amount of reservations for this size"`
	UsedReservations int      `json:"usedreservations" description:"the amount of used reservations for this size"`
	Faulty           int      `json:"faulty" description:"servers with issues with this size"`
	FaultyMachines   []string `json:"faultymachines" description:"servers with issues with this size"`
	Other            int      `json:"other" description:"servers neither free, allocated or faulty with this size"`
	OtherMachines    []string `json:"othermachines" description:"servers neither free, allocated or faulty with this size"`
}

func NewPartitionResponse(p *metal.Partition) *PartitionResponse {
	if p == nil {
		return nil
	}

	prefixLength := int(p.PrivateNetworkPrefixLength)

	labels := map[string]string{}
	if p.Labels != nil {
		labels = p.Labels
	}

	return &PartitionResponse{
		Common: Common{
			Identifiable: Identifiable{
				ID: p.ID,
			},
			Describable: Describable{
				Name:        &p.Name,
				Description: &p.Description,
			},
		},
		PartitionBase: PartitionBase{
			MgmtServiceAddress:          &p.MgmtServiceAddress,
			PrivateNetworkPrefixLength:  &prefixLength,
			PartitionWaitingPoolMinSize: &p.WaitingPoolMinSize,
			PartitionWaitingPoolMaxSize: &p.WaitingPoolMaxSize,
		},
		PartitionBootConfiguration: PartitionBootConfiguration{
			ImageURL:    &p.BootConfiguration.ImageURL,
			KernelURL:   &p.BootConfiguration.KernelURL,
			CommandLine: &p.BootConfiguration.CommandLine,
		},
		Timestamps: Timestamps{
			Created: p.Created,
			Changed: p.Changed,
		},
		Labels: labels,
	}
}

func (s ServerCapacities) FindBySize(size string) *ServerCapacity {
	for _, sc := range s {
		sc := sc
		if sc.Size == size {
			return sc
		}
	}

	return nil
}
