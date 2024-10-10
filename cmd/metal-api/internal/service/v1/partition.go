package v1

import (
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
)

type PartitionBase struct {
	MgmtServiceAddress         *string           `json:"mgmtserviceaddress" description:"the address to the management service of this partition" optional:"true"`
	PrivateNetworkPrefixLength *int              `json:"privatenetworkprefixlength" description:"the length of private networks for the machine's child networks in this partition, default 22" optional:"true" minimum:"16" maximum:"30"`
	Labels                     map[string]string `json:"labels" description:"free labels that you associate with this partition" optional:"true"`
}

type PartitionBootConfiguration struct {
	ImageURL                   *string                     `json:"imageurl" modelDescription:"a partition has a distinct location in a data center, individual entities belong to a partition" description:"the url to download the initrd for the boot image" optional:"true"`
	KernelURL                  *string                     `json:"kernelurl" description:"the url to download the kernel for the boot image" optional:"true"`
	CommandLine                *string                     `json:"commandline" description:"the cmdline to the kernel for the boot image" optional:"true"`
	MachineNetworkRequirements *MachineNetworkRequirements `json:"machine_network_requirements" description:"MachineNetworkRequirements defines the conditions required to take a machine into waiting state"`
}

// MachineNetworkRequirements defines the conditions required to take a machine into waiting state.
type MachineNetworkRequirements struct {
	MinimumInterfaces *int32 `json:"minimum_interfaces" description:"MinimumInterfaces definesdefines how many network cards must be detected with link on a machine to go into waiting state"`
	MinimumNeighbors  *int32 `json:"minimum_neighbors" description:"MinimumNeighbors defines how many distinct switch neighbors must be detected on a machine to go into waiting state"`
}

type PartitionCreateRequest struct {
	Common
	PartitionBase
	PartitionBootConfiguration PartitionBootConfiguration `json:"bootconfig" description:"the boot configuration of this partition"`
}

type PartitionUpdateRequest struct {
	Common
	MgmtServiceAddress         *string                     `json:"mgmtserviceaddress" description:"the address to the management service of this partition" optional:"true"`
	PartitionBootConfiguration *PartitionBootConfiguration `json:"bootconfig" description:"the boot configuration of this partition" optional:"true"`
	Labels                     map[string]string           `json:"labels" description:"free labels that you associate with this partition" optional:"true"`
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

// ServerCapacity holds the machine capacity of a partition of a specific size.
// The amount of allocated, waiting and other machines sum up to the total amount of machines.
type ServerCapacity struct {
	// Size is the size id correlating to all counts in this server capacity.
	Size string `json:"size" description:"the size of the machine"`

	// Total is the total amount of machines for this size.
	Total int `json:"total,omitempty" description:"total amount of machines with size"`

	// PhonedHome is the amount of machines that are currently in the provisioning state "phoned home".
	PhonedHome int `json:"phoned_home,omitempty" description:"machines in phoned home provisioning state"`
	// Waiting is the amount of machines that are currently in the provisioning state "waiting".
	Waiting int `json:"waiting,omitempty" description:"machines in waiting provisioning state"`
	// Other is the amount of machines that are neither in the provisioning state waiting nor in phoned home but in another provisioning state.
	Other int `json:"other,omitempty" description:"machines neither phoned home nor waiting but in another provisioning state"`
	// OtherMachines contains the machine IDs for machines that were classified into "Other".
	OtherMachines []string `json:"othermachines,omitempty" description:"machine ids neither allocated nor waiting with this size"`

	// Allocated is the amount of machines that are currently allocated.
	Allocated int `json:"allocated,omitempty" description:"allocated machines"`
	// Allocatable is the amount of machines in a partition is the amount of machines that can be allocated.
	// Effectively this is the amount of waiting machines minus the machines that are unavailable due to machine state or un-allocatable. Size reservations are not considered in this count.
	Allocatable int `json:"allocatable,omitempty" description:"free machines with this size, size reservations are not considered"`
	// Free is the amount of machines in a partition that can be freely allocated at any given moment by a project.
	// Effectively this is the amount of waiting machines minus the machines that are unavailable due to machine state or un-allocatable due to size reservations.
	Free int `json:"free,omitempty" description:"free machines with this size (freely allocatable)"`
	// Unavailable is the amount of machine in a partition that are currently not allocatable because they are not waiting or
	// not in the machine state "available", e.g. locked or reserved.
	Unavailable int `json:"unavailable,omitempty" description:"unavailable machines with this size"`

	// Faulty is the amount of machines that are neither allocated nor in the pool of available machines because they report an error.
	Faulty int `json:"faulty,omitempty" description:"machines with issues with this size"`
	// FaultyMachines contains the machine IDs for machines that were classified into "Faulty".
	FaultyMachines []string `json:"faultymachines,omitempty" description:"machine ids with issues with this size"`

	// Reservations is the amount of reservations made for this size.
	Reservations int `json:"reservations,omitempty" description:"the amount of reservations for this size"`
	// UsedReservations is the amount of reservations already used up for this size.
	UsedReservations int `json:"usedreservations,omitempty" description:"the amount of used reservations for this size"`
	// RemainingReservations is the amount of reservations remaining for this size.
	RemainingReservations int `json:"remainingreservations,omitempty" description:"the amount of unused / remaining / open reservations for this size"`
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

	var machineNetworkRequirements *MachineNetworkRequirements
	if p.BootConfiguration.MachineNetworkRequirements != nil {
		machineNetworkRequirements = &MachineNetworkRequirements{
			MinimumInterfaces: p.BootConfiguration.MachineNetworkRequirements.MinimumInterfaces,
			MinimumNeighbors:  p.BootConfiguration.MachineNetworkRequirements.MinimumNeighbors,
		}
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
			MgmtServiceAddress:         &p.MgmtServiceAddress,
			PrivateNetworkPrefixLength: &prefixLength,
		},
		PartitionBootConfiguration: PartitionBootConfiguration{
			ImageURL:                   &p.BootConfiguration.ImageURL,
			KernelURL:                  &p.BootConfiguration.KernelURL,
			CommandLine:                &p.BootConfiguration.CommandLine,
			MachineNetworkRequirements: machineNetworkRequirements,
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
