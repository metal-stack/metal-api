package v1

import (
	"time"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
)

// RecentProvisioningEventsLimit defines how many recent events are added to the MachineRecentProvisioningEvents struct
const RecentProvisioningEventsLimit = 5

type MachineBase struct {
	Partition                *PartitionListResponse          `json:"partition" modelDescription:"A machine representing a bare metal machine." description:"the partition assigned to this machine" readOnly:"true"`
	RackID                   string                          `json:"rackid" description:"the rack assigned to this machine" readOnly:"true"`
	Size                     *SizeListResponse               `json:"size" description:"the size of this machine" readOnly:"true"`
	Hardware                 MachineHardware                 `json:"hardware" description:"the hardware of this machine"`
	Allocation               *MachineAllocation              `json:"allocation" description:"the allocation data of an allocated machine"`
	State                    MachineState                    `json:"state" rethinkdb:"state" description:"the state of this machine"`
	Liveliness               string                          `json:"liveliness" description:"the liveliness of this machine"`
	RecentProvisioningEvents MachineRecentProvisioningEvents `json:"events" description:"recent events of this machine during provisioning"`
	Tags                     []string                        `json:"tags" description:"tags for this machine"`
}

type MachineAllocation struct {
	Created         time.Time          `json:"created" description:"the time when the machine was created"`
	Name            string             `json:"name" description:"the name of the machine"`
	Description     string             `json:"description,omitempty" description:"a description for this machine" optional:"true"`
	LastPing        time.Time          `json:"last_ping" description:"the timestamp of the last phone home call/ping from the machine" optional:"true" readOnly:"true"`
	Tenant          string             `json:"tenant" description:"the tenant that this machine is assigned to"`
	Project         string             `json:"project" description:"the project that this machine is assigned to" `
	Image           *ImageListResponse `json:"image" description:"the image assigned to this machine" readOnly:"true" optional:"true"`
	MachineNetworks []MachineNetwork   `json:"networks" description:"the networks of this machine"`
	Hostname        string             `json:"hostname" description:"the hostname which will be used when creating the machine"`
	SSHPubKeys      []string           `json:"ssh_pub_keys" description:"the public ssh keys to access the machine with"`
	UserData        string             `json:"user_data,omitempty" description:"userdata to execute post installation tasks" optional:"true"`
	ConsolePassword *string            `json:"console_password" description:"the console password which was generated while provisioning" optional:"true"`
}

type MachineNetwork struct {
	NetworkID string   `json:"networkid" description:"the networkID of the allocated machine in this vrf"`
	IPs       []string `json:"ips" description:"the ip addresses of the allocated machine in this vrf"`
	Vrf       uint     `json:"vrf" description:"the vrf of the allocated machine"`
	ASN       int64    `json:"asn" description:"ASN number for this network in the bgp configuration"`
}

type MachineHardware struct {
	Memory   uint64               `json:"memory" description:"the total memory of the machine"`
	CPUCores int                  `json:"cpu_cores" description:"the number of cpu cores"`
	Nics     MachineNics          `json:"nics" description:"the list of network interfaces of this machine"`
	Disks    []MachineBlockDevice `json:"disks" description:"the list of block devices of this machine"`
}

type MachineState struct {
	Value       string `json:"value" description:"the state of this machine. empty means available for all"`
	Description string `json:"description" description:"a description why this machine is in the given state"`
}

type MachineBlockDevice struct {
	Name string `json:"name" description:"the name of this block device"`
	Size uint64 `json:"size" description:"the size of this block device"`
}

type MachineRecentProvisioningEvents struct {
	Events                       []MachineProvisioningEvent `json:"log" description:"the log of recent machine provisioning events"`
	LastEventTime                *time.Time                 `json:"last_event_time" description:"the time where the last event was received"`
	IncompleteProvisioningCycles *string                    `json:"incomplete_provisioning_cycles" description:"the amount of incomplete provisioning cycles in the event container"`
}

type MachineProvisioningEvent struct {
	Time    time.Time `json:"time" description:"the time that this event was received" optional:"true" readOnly:"true"`
	Event   string    `json:"event" description:"the event emitted by the machine"`
	Message string    `json:"message" description:"an additional message to add to the event" optional:"true"`
}

type MachineNics []MachineNic

type MachineNic struct {
	MacAddress string      `json:"mac"  description:"the mac address of this network interface"`
	Name       string      `json:"name"  description:"the name of this network interface"`
	Neighbors  MachineNics `json:"neighbors" description:"the neighbors visible to this network interface"`
	// Vrf        string `json:"vrf" description:"the vrf this network interface is part of" optional:"true"` TODO: Possibly only part of switch nic
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
	UUID        string          `json:"uuid" description:"the product uuid of the machine to register"`
	PartitionID string          `json:"partitionid" description:"the partition id to register this machine with"`
	RackID      string          `json:"rackid" description:"the rack id where this machine is connected to"`
	Hardware    MachineHardware `json:"hardware" description:"the hardware of this machine"`
	IPMI        MachineIPMI     `json:"ipmi" description:"the ipmi access infos"`
	Tags        []string        `json:"tags" description:"tags for this machine"`
}

type MachineAllocateRequest struct {
	UUID string `json:"uuid,omitempty" description:"if this field is set, this specific machine will be allocated if it is not in available state and not currently allocated. this field overrules size and partition"`
	Describeable
	Tenant      string   `json:"tenant" description:"the name of the owning tenant"`
	Hostname    string   `json:"hostname" description:"the hostname for the allocated machine"`
	ProjectID   string   `json:"projectid" description:"the project id to assign this machine to"`
	PartitionID string   `json:"partitionid" description:"the partition id to assign this machine to"`
	SizeID      string   `json:"sizeid" description:"the size id to assign this machine to"`
	ImageID     string   `json:"imageid" description:"the image id to assign this machine to"`
	SSHPubKeys  []string `json:"ssh_pub_keys" description:"the public ssh keys to access the machine with"`
	UserData    string   `json:"user_data,omitempty" description:"cloud-init.io compatible userdata must be base64 encoded." optional:"true" rethinkdb:"userdata"`
	Tags        []string `json:"tags" description:"tags for this machine" rethinkdb:"tags"`
	NetworkIDs  []string `json:"networks" description:"the networks of this firewall, required."`
	IPs         []string `json:"ips" description:"the additional ips of this firewall, optional."`
	HA          bool     `json:"ha" description:"if set to true, this firewall is set up in a High Available manner" optional:"true"`
}

type MachineFinalizeAllocationRequest struct {
	ConsolePassword string `json:"console_password" description:"the console password which was generated while provisioning" optional:"false"`
}

type MachineWaitResponse struct {
	MachineDetailResponse
	PhoneHomeToken string `json:"phone_home_token"`
}

type MachineListResponse struct {
	Common
	MachineBase
}

type MachineDetailResponse struct {
	MachineListResponse
	Timestamps
}

func NewMetalMachineHardware(r *MachineHardware) metal.MachineHardware {
	var nics metal.Nics
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

func NewMachineDetailResponse(m *metal.Machine, s *metal.Size, p *metal.Partition, i *metal.Image, ec *metal.ProvisioningEventContainer) *MachineDetailResponse {
	return &MachineDetailResponse{
		MachineListResponse: *NewMachineListResponse(m, s, p, i, ec),
		Timestamps: Timestamps{
			Created: m.Created,
			Changed: m.Changed,
		},
	}
}

func NewMachineListResponse(m *metal.Machine, s *metal.Size, p *metal.Partition, i *metal.Image, ec *metal.ProvisioningEventContainer) *MachineListResponse {
	var hardware MachineHardware
	var nics MachineNics
	for i := range m.Hardware.Nics {
		// TODO: Do we need to return the neighbors here for the clients?
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
			Memory:   m.Hardware.Memory,
			CPUCores: m.Hardware.CPUCores,
			Nics:     nics,
			Disks:    disks,
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
				NetworkID: m.Allocation.MachineNetworks[i].NetworkID,
				IPs:       ips,
				Vrf:       m.Allocation.MachineNetworks[i].Vrf,
				ASN:       m.Allocation.MachineNetworks[i].ASN,
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
			LastPing:        m.Allocation.LastPing,
			Image:           NewImageListResponse(i),
			Tenant:          m.Allocation.Tenant,
			Project:         m.Allocation.Project,
			Hostname:        m.Allocation.Hostname,
			SSHPubKeys:      m.Allocation.SSHPubKeys,
			UserData:        m.Allocation.UserData,
			ConsolePassword: consolePassword,
			MachineNetworks: networks,
		}
	}

	tags := []string{}
	if len(tags) > 0 {
		tags = m.Tags
	}

	return &MachineListResponse{
		Common: Common{
			Identifiable: Identifiable{
				ID: m.ID,
			},
			Describeable: Describeable{
				Name:        &m.Name,
				Description: &m.Description,
			},
		},
		MachineBase: MachineBase{
			Partition:  NewPartitionListResponse(p),
			Size:       NewSizeListResponse(s),
			Allocation: allocation,
			RackID:     m.RackID,
			Hardware:   hardware,
			State: MachineState{
				Value:       string(m.State.Value),
				Description: m.State.Description,
			},
			Liveliness:               string(m.Liveliness),
			RecentProvisioningEvents: *NewMachineRecentProvisioningEvents(ec),
			Tags:                     tags,
		},
	}
}

func NewMachineRecentProvisioningEvents(ec *metal.ProvisioningEventContainer) *MachineRecentProvisioningEvents {
	es := []MachineProvisioningEvent{}
	if ec == nil {
		return &MachineRecentProvisioningEvents{
			Events:                       es,
			LastEventTime:                nil,
			IncompleteProvisioningCycles: nil,
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
	var incompleteCycles *string
	if ec.IncompleteProvisioningCycles != "" {
		incompleteCycles = &ec.IncompleteProvisioningCycles
	}
	return &MachineRecentProvisioningEvents{
		Events:                       es,
		IncompleteProvisioningCycles: incompleteCycles,
		LastEventTime:                ec.LastEventTime,
	}
}

func NewMachineWaitResponse(m *metal.Machine, s *metal.Size, p *metal.Partition, i *metal.Image, ec *metal.ProvisioningEventContainer, token string) *MachineWaitResponse {
	return &MachineWaitResponse{
		MachineDetailResponse: *NewMachineDetailResponse(m, s, p, i, ec),
		PhoneHomeToken:        m.Allocation.PhoneHomeToken,
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
