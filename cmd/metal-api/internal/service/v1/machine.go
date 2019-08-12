package v1

import (
	"time"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
)

// RecentProvisioningEventsLimit defines how many recent events are added to the MachineRecentProvisioningEvents struct
const RecentProvisioningEventsLimit = 5

type MachineBase struct {
	Partition                *PartitionResponse              `json:"partition" modelDescription:"A machine representing a bare metal machine." description:"the partition assigned to this machine" readOnly:"true"`
	RackID                   string                          `json:"rackid" description:"the rack assigned to this machine" readOnly:"true"`
	Size                     *SizeResponse                   `json:"size" description:"the size of this machine" readOnly:"true"`
	Hardware                 MachineHardware                 `json:"hardware" description:"the hardware of this machine"`
	Allocation               *MachineAllocation              `json:"allocation" description:"the allocation data of an allocated machine"`
	State                    MachineState                    `json:"state" rethinkdb:"state" description:"the state of this machine"`
	LEDState                 MachineLEDState                 `json:"ledstate" rethinkdb:"ledstate" description:"the state of this machine chassis identify LED"`
	Liveliness               string                          `json:"liveliness" description:"the liveliness of this machine"`
	RecentProvisioningEvents MachineRecentProvisioningEvents `json:"events" description:"recent events of this machine during provisioning"`
	Tags                     []string                        `json:"tags" description:"tags for this machine"`
}

type MachineAllocation struct {
	Created         time.Time        `json:"created" description:"the time when the machine was created"`
	Name            string           `json:"name" description:"the name of the machine"`
	Description     string           `json:"description,omitempty" description:"a description for this machine" optional:"true"`
	Tenant          string           `json:"tenant" description:"the tenant that this machine is assigned to"`
	Project         string           `json:"project" description:"the project that this machine is assigned to" `
	Image           *ImageResponse   `json:"image" description:"the image assigned to this machine" readOnly:"true" optional:"true"`
	MachineNetworks []MachineNetwork `json:"networks" description:"the networks of this machine"`
	Hostname        string           `json:"hostname" description:"the hostname which will be used when creating the machine"`
	SSHPubKeys      []string         `json:"ssh_pub_keys" description:"the public ssh keys to access the machine with"`
	UserData        string           `json:"user_data,omitempty" description:"userdata to execute post installation tasks" optional:"true"`
	ConsolePassword *string          `json:"console_password" description:"the console password which was generated while provisioning" optional:"true"`
	Succeeded       bool             `json:"succeeded" description:"if the allocation of the machine was successful, this is set to true"`
}

type MachineNetwork struct {
	NetworkID           string   `json:"networkid" description:"the networkID of the allocated machine in this vrf"`
	Prefixes            []string `json:"prefixes" description:"the prefixes of this network"`
	IPs                 []string `json:"ips" description:"the ip addresses of the allocated machine in this vrf"`
	Vrf                 uint     `json:"vrf" description:"the vrf of the allocated machine"`
	ASN                 int64    `json:"asn" description:"ASN number for this network in the bgp configuration"`
	Primary             bool     `json:"primary" description:"indicates whether this network is the primary network of this machine"`
	Nat                 bool     `json:"nat" description:"if set to true, packets leaving this network get masqueraded behind interface ip"`
	DestinationPrefixes []string `json:"destinationprefixes" modelDescription:"prefixes that are reachable within this network" description:"the destination prefixes of this network"`
	Underlay            bool     `json:"underlay" description:"if set to true, this network can be used for underlay communication"`
}

type MachineHardwareBase struct {
	Memory   uint64               `json:"memory" description:"the total memory of the machine"`
	CPUCores int                  `json:"cpu_cores" description:"the number of cpu cores"`
	Disks    []MachineBlockDevice `json:"disks" description:"the list of block devices of this machine"`
}

type MachineHardware struct {
	MachineHardwareBase
	Nics MachineNics `json:"nics" description:"the list of network interfaces of this machine"`
}

type MachineHardwareExtended struct {
	MachineHardwareBase
	Nics MachineNicsExtended `json:"nics" description:"the list of network interfaces of this machine with extended information"`
}

type MachineState struct {
	Value       string `json:"value" description:"the state of this machine. empty means available for all"`
	Description string `json:"description" description:"a description why this machine is in the given state"`
}

type MachineLEDState struct {
	Value       string `json:"value" description:"the state of this machine chassis identify LED. empty means On"`
	Description string `json:"description" description:"a description why this machine chassis identify LED is in the given state"`
}

type MachineBlockDevice struct {
	Name string `json:"name" description:"the name of this block device"`
	Size uint64 `json:"size" description:"the size of this block device"`
}

type MachineRecentProvisioningEvents struct {
	Events                       []MachineProvisioningEvent `json:"log" description:"the log of recent machine provisioning events"`
	LastEventTime                *time.Time                 `json:"last_event_time" description:"the time where the last event was received"`
	IncompleteProvisioningCycles string                     `json:"incomplete_provisioning_cycles" description:"the amount of incomplete provisioning cycles in the event container"`
}

type MachineProvisioningEvent struct {
	Time    time.Time `json:"time" description:"the time that this event was received" optional:"true" readOnly:"true"`
	Event   string    `json:"event" description:"the event emitted by the machine"`
	Message string    `json:"message" description:"an additional message to add to the event" optional:"true"`
}

type MachineNics []MachineNic

type MachineNic struct {
	MacAddress string `json:"mac"  description:"the mac address of this network interface"`
	Name       string `json:"name"  description:"the name of this network interface"`
}

type MachineNicsExtended []MachineNicExtended

type MachineNicExtended struct {
	MachineNic
	Neighbors MachineNicsExtended `json:"neighbors" description:"the neighbors visible to this network interface"`
}

type MachineLivelinessReport struct {
	AliveCount   int `json:"alive_count" description:"the number of machines alive"`
	DeadCount    int `json:"dead_count" description:"the number of dead machines"`
	UnknownCount int `json:"unknown_count" description:"the number of machines with unknown liveliness"`
}

type MachineIPMI struct {
	Address    string     `json:"address" modelDescription:"The IPMI connection data"`
	MacAddress string     `json:"mac"`
	User       string     `json:"user"`
	Password   string     `json:"password"`
	Interface  string     `json:"interface"`
	Fru        MachineFru `json:"fru" `
}

type MachineFru struct {
	ChassisPartNumber   *string `json:"chassis_part_number,omitempty" modelDescription:"The Field Replaceable Unit data" description:"the chassis part number" optional:"true"`
	ChassisPartSerial   *string `json:"chassis_part_serial,omitempty" description:"the chassis part serial" optional:"true"`
	BoardMfg            *string `json:"board_mfg,omitempty" description:"the board mfg" optional:"true"`
	BoardMfgSerial      *string `json:"board_mfg_serial,omitempty" description:"the board mfg serial" optional:"true"`
	BoardPartNumber     *string `json:"board_part_number,omitempty" description:"the board part number" optional:"true"`
	ProductManufacturer *string `json:"product_manufacturer,omitempty" description:"the product manufacturer" optional:"true"`
	ProductPartNumber   *string `json:"product_part_number,omitempty" description:"the product part number" optional:"true"`
	ProductSerial       *string `json:"product_serial,omitempty" description:"the product serial" optional:"true"`
}

type MachineRegisterRequest struct {
	UUID        string                  `json:"uuid" description:"the product uuid of the machine to register"`
	PartitionID string                  `json:"partitionid" description:"the partition id to register this machine with"`
	RackID      string                  `json:"rackid" description:"the rack id where this machine is connected to"`
	Hardware    MachineHardwareExtended `json:"hardware" description:"the hardware of this machine"`
	IPMI        MachineIPMI             `json:"ipmi" description:"the ipmi access infos"`
	Tags        []string                `json:"tags" description:"tags for this machine"`
}

type MachineAllocateRequest struct {
	UUID *string `json:"uuid" description:"if this field is set, this specific machine will be allocated if it is not in available state and not currently allocated. this field overrules size and partition" optional:"true"`
	Describable
	Tenant      string                    `json:"tenant" description:"the name of the owning tenant"`
	Hostname    *string                   `json:"hostname" description:"the hostname for the allocated machine (defaults to metal)" optional:"true"`
	ProjectID   string                    `json:"projectid" description:"the project id to assign this machine to"`
	PartitionID string                    `json:"partitionid" description:"the partition id to assign this machine to"`
	SizeID      string                    `json:"sizeid" description:"the size id to assign this machine to"`
	ImageID     string                    `json:"imageid" description:"the image id to assign this machine to"`
	SSHPubKeys  []string                  `json:"ssh_pub_keys" description:"the public ssh keys to access the machine with"`
	UserData    *string                   `json:"user_data" description:"cloud-init.io compatible userdata must be base64 encoded" optional:"true"`
	Tags        []string                  `json:"tags" description:"tags for this machine" optional:"true"`
	Networks    MachineAllocationNetworks `json:"networks" description:"the networks that this machine will be placed in." optional:"true"`
	IPs         []string                  `json:"ips" description:"the ips to attach to this machine additionally" optional:"true"`
}

type MachineAllocationNetworks []MachineAllocationNetwork

type MachineAllocationNetwork struct {
	NetworkID     string `json:"networkid" description:"the id of the network that this machine will be placed in"`
	AutoAcquireIP *bool  `json:"autoacquire" description:"will automatically acquire an ip in this network if set to true, default is true"`
}

type MachineFinalizeAllocationRequest struct {
	ConsolePassword string `json:"console_password" description:"the console password which was generated while provisioning"`
}

type FindMachinesRequest struct {
	ID          *string  `json:"id"`
	Name        *string  `json:"name"`
	PartitionID *string  `json:"partition_id"`
	SizeID      *string  `json:"sizeid"`
	RackID      *string  `json:"rackid"`
	Liveliness  *string  `json:"liveliness"`
	Tags        []string `json:"tags"`

	// allocation
	AllocationName      *string `json:"allocation_name"`
	AllocationTenant    *string `json:"allocation_tenant"`
	AllocationProject   *string `json:"allocation_project"`
	AllocationImageID   *string `json:"allocation_image_id"`
	AllocationHostname  *string `json:"allocation_hostname"`
	AllocationSucceeded *bool   `json:"allocation_succeeded"`

	// network
	NetworkIDs                 []string `json:"network_ids"`
	NetworkPrefixes            []string `json:"network_prefixes"`
	NetworkIPs                 []string `json:"network_ips"`
	NetworkDestinationPrefixes []string `json:"network_destination_prefixes"`
	NetworkVrfs                []int64  `json:"network_vrfs"`
	NetworkPrimary             *bool    `json:"network_primary"`
	NetworkASNs                []int64  `json:"network_asns"`
	NetworkNat                 *bool    `json:"network_nat"`
	NetworkUnderlay            *bool    `json:"network_underlay"`

	// hardware
	HardwareMemory   *int64 `json:"hardware_memory"`
	HardwareCPUCores *int64 `json:"hardware_cpu_cores"`

	// nics
	NicsMacAddresses         []string `json:"nics_mac_addresses"`
	NicsNames                []string `json:"nics_names"`
	NicsVrfs                 []string `json:"nics_vrfs"`
	NicsNeighborMacAddresses []string `json:"nics_neighbor_mac_addresses"`
	NicsNeighborNames        []string `json:"nics_neighbor_names"`
	NicsNeighborVrfs         []string `json:"nics_neighbor_vrfs"`

	// disks
	DiskNames []string `json:"disk_names"`
	DiskSizes []int64  `json:"disk_sizes"`

	// state
	StateValue *string `json:"state_value"`

	// ipmi
	IpmiAddress    *string `json:"ipmi_address"`
	IpmiMacAddress *string `json:"ipmi_mac_address"`
	IpmiUser       *string `json:"ipmi_user"`
	IpmiInterface  *string `json:"ipmi_interface"`

	// fru
	FruChassisPartNumber   *string `json:"fru_chassis_part_number"`
	FruChassisPartSerial   *string `json:"fru_chassis_part_serial"`
	FruBoardMfg            *string `json:"fru_board_mfg"`
	FruBoardMfgSerial      *string `json:"fru_board_mfg_serial"`
	FruBoardPartNumber     *string `json:"fru_board_part_number"`
	FruProductManufacturer *string `json:"fru_product_manufacturer"`
	FruProductPartNumber   *string `json:"fru_product_part_number"`
	FruProductSerial       *string `json:"fru_product_serial"`
}

type MachineResponse struct {
	Common
	MachineBase
	Timestamps
}

func NewMetalMachineHardware(r *MachineHardwareExtended) metal.MachineHardware {
	nics := metal.Nics{}
	for i := range r.Nics {
		var neighbors metal.Nics
		for i2 := range r.Nics[i].Neighbors {
			neighbor := metal.Nic{
				MacAddress: metal.MacAddress(r.Nics[i].Neighbors[i2].MacAddress),
				Name:       r.Nics[i].Neighbors[i2].Name,
			}
			neighbors = append(neighbors, neighbor)
		}
		nic := metal.Nic{
			MacAddress: metal.MacAddress(r.Nics[i].MacAddress),
			Name:       r.Nics[i].Name,
			Neighbors:  neighbors,
		}
		nics = append(nics, nic)
	}
	var disks []metal.BlockDevice
	for i := range r.Disks {
		disk := metal.BlockDevice{
			Name: r.Disks[i].Name,
			Size: r.Disks[i].Size,
		}
		disks = append(disks, disk)
	}
	return metal.MachineHardware{
		Memory:   r.Memory,
		CPUCores: r.CPUCores,
		Nics:     nics,
		Disks:    disks,
	}
}

func NewMetalIPMI(r *MachineIPMI) metal.IPMI {
	var chassisPartNumber string
	if r.Fru.ChassisPartNumber != nil {
		chassisPartNumber = *r.Fru.ChassisPartNumber
	}
	var chassisPartSerial string
	if r.Fru.ChassisPartSerial != nil {
		chassisPartSerial = *r.Fru.ChassisPartSerial
	}
	var boardMfg string
	if r.Fru.BoardMfg != nil {
		boardMfg = *r.Fru.BoardMfg
	}
	var boardMfgSerial string
	if r.Fru.BoardMfgSerial != nil {
		boardMfgSerial = *r.Fru.BoardMfgSerial
	}
	var boardPartNumber string
	if r.Fru.BoardPartNumber != nil {
		boardPartNumber = *r.Fru.BoardPartNumber
	}
	var productManufacturer string
	if r.Fru.ProductManufacturer != nil {
		productManufacturer = *r.Fru.ProductManufacturer
	}
	var productPartNumber string
	if r.Fru.ProductPartNumber != nil {
		productPartNumber = *r.Fru.ProductPartNumber
	}
	var productSerial string
	if r.Fru.ProductSerial != nil {
		productSerial = *r.Fru.ProductSerial
	}

	return metal.IPMI{
		Address:    r.Address,
		MacAddress: r.MacAddress,
		User:       r.User,
		Password:   r.Password,
		Interface:  r.Interface,
		Fru: metal.Fru{
			ChassisPartNumber:   chassisPartNumber,
			ChassisPartSerial:   chassisPartSerial,
			BoardMfg:            boardMfg,
			BoardMfgSerial:      boardMfgSerial,
			BoardPartNumber:     boardPartNumber,
			ProductManufacturer: productManufacturer,
			ProductPartNumber:   productPartNumber,
			ProductSerial:       productSerial,
		},
	}
}

func NewMachineResponse(m *metal.Machine, s *metal.Size, p *metal.Partition, i *metal.Image, ec *metal.ProvisioningEventContainer) *MachineResponse {
	var hardware MachineHardware
	nics := MachineNics{}
	for i := range m.Hardware.Nics {
		nic := MachineNic{
			MacAddress: string(m.Hardware.Nics[i].MacAddress),
			Name:       m.Hardware.Nics[i].Name,
		}
		nics = append(nics, nic)

		var disks []MachineBlockDevice
		for i := range m.Hardware.Disks {
			disk := MachineBlockDevice{
				Name: m.Hardware.Disks[i].Name,
				Size: m.Hardware.Disks[i].Size,
			}
			disks = append(disks, disk)
		}
		hardware = MachineHardware{
			MachineHardwareBase: MachineHardwareBase{
				Memory:   m.Hardware.Memory,
				CPUCores: m.Hardware.CPUCores,
				Disks:    disks,
			},
			Nics: nics,
		}
	}

	var allocation *MachineAllocation
	if m.Allocation != nil {
		networks := []MachineNetwork{}
		for i := range m.Allocation.MachineNetworks {
			ips := []string{}
			for _, ip := range m.Allocation.MachineNetworks[i].IPs {
				ips = append(ips, ip)
			}
			network := MachineNetwork{
				NetworkID:           m.Allocation.MachineNetworks[i].NetworkID,
				IPs:                 ips,
				Vrf:                 m.Allocation.MachineNetworks[i].Vrf,
				ASN:                 m.Allocation.MachineNetworks[i].ASN,
				Primary:             m.Allocation.MachineNetworks[i].Primary,
				Nat:                 m.Allocation.MachineNetworks[i].Nat,
				Underlay:            m.Allocation.MachineNetworks[i].Underlay,
				DestinationPrefixes: m.Allocation.MachineNetworks[i].DestinationPrefixes,
				Prefixes:            m.Allocation.MachineNetworks[i].Prefixes,
			}
			networks = append(networks, network)
		}

		var consolePassword *string
		if m.Allocation.ConsolePassword != "" {
			consolePassword = &m.Allocation.ConsolePassword
		}

		allocation = &MachineAllocation{
			Created:         m.Allocation.Created,
			Name:            m.Allocation.Name,
			Description:     m.Allocation.Description,
			Image:           NewImageResponse(i),
			Tenant:          m.Allocation.Tenant,
			Project:         m.Allocation.Project,
			Hostname:        m.Allocation.Hostname,
			SSHPubKeys:      m.Allocation.SSHPubKeys,
			UserData:        m.Allocation.UserData,
			ConsolePassword: consolePassword,
			MachineNetworks: networks,
			Succeeded:       m.Allocation.Succeeded,
		}
	}

	tags := []string{}
	if len(m.Tags) > 0 {
		tags = m.Tags
	}

	return &MachineResponse{
		Common: Common{
			Identifiable: Identifiable{
				ID: m.ID,
			},
			Describable: Describable{
				Name:        &m.Name,
				Description: &m.Description,
			},
		},
		MachineBase: MachineBase{
			Partition:  NewPartitionResponse(p),
			Size:       NewSizeResponse(s),
			Allocation: allocation,
			RackID:     m.RackID,
			Hardware:   hardware,
			State: MachineState{
				Value:       string(m.State.Value),
				Description: m.State.Description,
			},
			LEDState: MachineLEDState{
				Value:       string(m.LEDState.Value),
				Description: m.LEDState.Description,
			},
			Liveliness:               string(ec.Liveliness),
			RecentProvisioningEvents: *NewMachineRecentProvisioningEvents(ec),
			Tags:                     tags,
		},
		Timestamps: Timestamps{
			Created: m.Created,
			Changed: m.Changed,
		},
	}
}

func NewMachineRecentProvisioningEvents(ec *metal.ProvisioningEventContainer) *MachineRecentProvisioningEvents {
	es := []MachineProvisioningEvent{}
	if ec == nil {
		return &MachineRecentProvisioningEvents{
			Events:                       es,
			LastEventTime:                nil,
			IncompleteProvisioningCycles: "0",
		}
	}
	machineEvents := ec.Events
	if len(machineEvents) >= RecentProvisioningEventsLimit {
		machineEvents = machineEvents[:RecentProvisioningEventsLimit]
	}
	for _, machineEvent := range machineEvents {
		e := MachineProvisioningEvent{
			Time:    machineEvent.Time,
			Event:   string(machineEvent.Event),
			Message: machineEvent.Message,
		}
		es = append(es, e)
	}
	return &MachineRecentProvisioningEvents{
		Events:                       es,
		IncompleteProvisioningCycles: ec.IncompleteProvisioningCycles,
		LastEventTime:                ec.LastEventTime,
	}
}

func NewMachineIPMI(m *metal.IPMI) *MachineIPMI {
	if m == nil {
		return &MachineIPMI{}
	}
	return &MachineIPMI{
		Address:    m.Address,
		MacAddress: m.MacAddress,
		User:       m.User,
		Password:   m.Password,
		Interface:  m.Interface,
		Fru: MachineFru{
			ChassisPartNumber:   &m.Fru.ChassisPartNumber,
			ChassisPartSerial:   &m.Fru.ChassisPartSerial,
			BoardMfg:            &m.Fru.BoardMfg,
			BoardMfgSerial:      &m.Fru.BoardMfgSerial,
			BoardPartNumber:     &m.Fru.BoardPartNumber,
			ProductManufacturer: &m.Fru.ProductManufacturer,
			ProductPartNumber:   &m.Fru.ProductPartNumber,
			ProductSerial:       &m.Fru.ProductSerial,
		},
	}
}
