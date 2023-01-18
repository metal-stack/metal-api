package v1

import (
	"time"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
)

// RecentProvisioningEventsLimit defines how many recent events are added to the MachineRecentProvisioningEvents struct
const RecentProvisioningEventsLimit = 20

type MachineBase struct {
	Partition                *PartitionResponse              `json:"partition" modelDescription:"A machine representing a bare metal machine." description:"the partition assigned to this machine" readOnly:"true" optional:"true"`
	RackID                   string                          `json:"rackid" description:"the rack assigned to this machine" readOnly:"true" optional:"true"`
	Size                     *SizeResponse                   `json:"size" description:"the size of this machine" readOnly:"true" optional:"true"`
	Hardware                 MachineHardware                 `json:"hardware" description:"the hardware of this machine"`
	BIOS                     MachineBIOS                     `json:"bios" description:"bios information of this machine"`
	Allocation               *MachineAllocation              `json:"allocation" description:"the allocation data of an allocated machine" optional:"true"`
	State                    MachineState                    `json:"state" rethinkdb:"state" description:"the state of this machine"`
	LEDState                 ChassisIdentifyLEDState         `json:"ledstate" rethinkdb:"ledstate" description:"the state of this chassis identify LED"`
	Liveliness               string                          `json:"liveliness" description:"the liveliness of this machine"`
	RecentProvisioningEvents MachineRecentProvisioningEvents `json:"events" description:"recent events of this machine during provisioning"`
	Tags                     []string                        `json:"tags" description:"tags for this machine"`
}

type MachineAllocation struct {
	Creator          string                    `json:"creator" description:"email of machine creator"`
	Created          time.Time                 `json:"created" description:"the time when the machine was created"`
	Name             string                    `json:"name" description:"the name of the machine"`
	Description      string                    `json:"description,omitempty" description:"a description for this machine" optional:"true"`
	Project          string                    `json:"project" description:"the project id that this machine is assigned to" `
	Image            *ImageResponse            `json:"image" description:"the image assigned to this machine" readOnly:"true" optional:"true"`
	FilesystemLayout *FilesystemLayoutResponse `json:"filesystemlayout" description:"filesystemlayout to create on this machine" optional:"true"`
	MachineNetworks  []MachineNetwork          `json:"networks" description:"the networks of this machine"`
	Hostname         string                    `json:"hostname" description:"the hostname which will be used when creating the machine"`
	SSHPubKeys       []string                  `json:"ssh_pub_keys" description:"the public ssh keys to access the machine with"`
	UserData         string                    `json:"user_data,omitempty" description:"userdata to execute post installation tasks" optional:"true"`
	Succeeded        bool                      `json:"succeeded" description:"if the allocation of the machine was successful, this is set to true"`
	Reinstall        bool                      `json:"reinstall" description:"indicates whether to reinstall the machine"`
	BootInfo         *BootInfo                 `json:"boot_info" description:"information required for booting the machine from HD" optional:"true"`
	Role             string                    `json:"role" enum:"machine|firewall" description:"the role of the machine"`
	VPN              *MachineVPN               `json:"vpn" description:"vpn connection info for machine" optional:"true"`
}

type BootInfo struct {
	ImageID      string `json:"image_id" description:"the ID of the current image"`
	PrimaryDisk  string `json:"primary_disk" description:"the primary disk"`
	OSPartition  string `json:"os_partition" description:"the partition containing the OS"`
	Initrd       string `json:"initrd" description:"the initrd image"`
	Cmdline      string `json:"cmdline" description:"the cmdline"`
	Kernel       string `json:"kernel" description:"the kernel"`
	BootloaderID string `json:"bootloaderid" description:"the bootloader ID"`
}

type MachineNetwork struct {
	NetworkID           string   `json:"networkid" description:"the networkID of the allocated machine in this vrf"`
	Prefixes            []string `json:"prefixes" description:"the prefixes of this network"`
	IPs                 []string `json:"ips" description:"the ip addresses of the allocated machine in this vrf"`
	DestinationPrefixes []string `json:"destinationprefixes" modelDescription:"prefixes that are reachable within this network" description:"the destination prefixes of this network"`
	NetworkType         string   `json:"networktype" description:"the network type, types can be looked up in the network package of metal-lib"`
	Vrf                 uint     `json:"vrf" description:"the vrf of the allocated machine"`
	// Attention, uint32 is converted to integer by swagger which is int32 which is to small to hold a asn
	ASN int64 `json:"asn" description:"ASN number for this network in the bgp configuration"`
	Nat bool  `json:"nat" description:"if set to true, packets leaving this network get masqueraded behind interface ip"`
	// Private flag to indicate this is a private network
	//
	// Deprecated: can be removed once old machine images without NetworkType are not supported anymore
	Private bool `json:"private" description:"indicates whether this network is the private network of this machine"`
	// Underlay flag to indicate this is a underlay network
	//
	// Deprecated: can be removed once old machine images without NetworkType are not supported anymore
	Underlay bool `json:"underlay" description:"if set to true, this network can be used for underlay communication"`
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

type MachineState struct {
	Value              string `json:"value" enum:"RESERVED|LOCKED|" description:"the state of this machine. empty means available for all"`
	Description        string `json:"description" description:"a description why this machine is in the given state"`
	Issuer             string `json:"issuer,omitempty" optional:"true" description:"the user that changed the state"`
	MetalHammerVersion string `json:"metal_hammer_version" description:"the version of metal hammer which put the machine in waiting state"`
}

type ChassisIdentifyLEDState struct {
	Value       string `json:"value" description:"the state of this chassis identify LED. empty means LED-OFF"`
	Description string `json:"description" description:"a description why this chassis identify LED is in the given state"`
}

type MachineBlockDevice struct {
	Name string `json:"name" description:"the name of this block device"`
	Size uint64 `json:"size" description:"the size of this block device"`
}

type MachineRecentProvisioningEvents struct {
	Events               []MachineProvisioningEvent `json:"log" description:"the log of recent machine provisioning events"`
	LastEventTime        *time.Time                 `json:"last_event_time" description:"the time where the last event was received" optional:"true"`
	LastErrorEvent       *MachineProvisioningEvent  `json:"last_error_event,omitempty" description:"the last erroneous event received" optional:"true"`
	CrashLoop            bool                       `json:"crash_loop" description:"indicates that machine is provisioning crash loop"`
	FailedMachineReclaim bool                       `json:"failed_machine_reclaim" description:"indicates that machine reclaim has failed"`
}

type MachineRecentProvisioningEventsResponse struct {
	Events uint64   `json:"events" description:"number of events stored"`
	Failed []string `json:"failed" description:"slice of machineIDs for which event was not published"`
}

type MachineProvisioningEvent struct {
	Time    time.Time `json:"time" description:"the time that this event was received" optional:"true" readOnly:"true"`
	Event   string    `json:"event" description:"the event emitted by the machine"`
	Message string    `json:"message" description:"an additional message to add to the event" optional:"true"`
}

type MachineProvisioningEvents map[string]MachineProvisioningEvent

type MachineNics []MachineNic

type MachineNic struct {
	MacAddress string      `json:"mac"  description:"the mac address of this network interface"`
	Name       string      `json:"name"  description:"the name of this network interface"`
	Neighbors  MachineNics `json:"neighbors" description:"the neighbors visible to this network interface"`
}

type MachineBIOS struct {
	Version string `json:"version" modelDescription:"The bios version" description:"the bios version"`
	Vendor  string `json:"vendor" description:"the bios vendor"`
	Date    string `json:"date" description:"the bios date"`
}

type MachineIPMI struct {
	Address     string       `json:"address" modelDescription:"The IPMI connection data"`
	MacAddress  string       `json:"mac"`
	User        string       `json:"user"`
	Password    string       `json:"password"`
	Interface   string       `json:"interface"`
	Fru         MachineFru   `json:"fru"`
	BMCVersion  string       `json:"bmcversion"`
	PowerState  string       `json:"powerstate"`
	PowerMetric *PowerMetric `json:"powermetric"`
}

type PowerMetric struct {
	// AverageConsumedWatts shall represent the
	// average power level that occurred averaged over the last IntervalInMin
	// minutes.
	AverageConsumedWatts float32 `json:"averageconsumedwatts"`
	// IntervalInMin shall represent the time
	// interval (or window), in minutes, in which the PowerMetrics properties
	// are measured over.
	// Should be an integer, but some Dell implementations return as a float.
	IntervalInMin float32 `json:"intervalinmin"`
	// MaxConsumedWatts shall represent the
	// maximum power level in watts that occurred within the last
	// IntervalInMin minutes.
	MaxConsumedWatts float32 `json:"maxconsumedwatts"`
	// MinConsumedWatts shall represent the
	// minimum power level in watts that occurred within the last
	// IntervalInMin minutes.
	MinConsumedWatts float32 `json:"minconsumedwatts"`
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

type MachineAllocateRequest struct {
	UUID *string `json:"uuid" description:"if this field is set, this specific machine will be allocated if it is not in available state and not currently allocated. this field overrules size and partition" optional:"true"`
	Describable
	Hostname           *string                   `json:"hostname" description:"the hostname for the allocated machine (defaults to metal)" optional:"true"`
	ProjectID          string                    `json:"projectid" description:"the project id to assign this machine to"`
	PartitionID        string                    `json:"partitionid" description:"the partition id to assign this machine to"`
	SizeID             string                    `json:"sizeid" description:"the size id to assign this machine to"`
	ImageID            string                    `json:"imageid" description:"the image id to assign this machine to"`
	FilesystemLayoutID *string                   `json:"filesystemlayoutid" description:"the filesystemlayout id to assing to this machine" optional:"true"`
	SSHPubKeys         []string                  `json:"ssh_pub_keys" description:"the public ssh keys to access the machine with"`
	UserData           *string                   `json:"user_data" description:"cloud-init.io compatible userdata must be base64 encoded" optional:"true"`
	Tags               []string                  `json:"tags" description:"tags for this machine" optional:"true"`
	Networks           MachineAllocationNetworks `json:"networks" description:"the networks that this machine will be placed in." optional:"true"`
	IPs                []string                  `json:"ips" description:"the ips to attach to this machine additionally" optional:"true"`
}

type MachineAllocationNetworks []MachineAllocationNetwork

type MachineAllocationNetwork struct {
	NetworkID     string `json:"networkid" description:"the id of the network that this machine will be placed in"`
	AutoAcquireIP *bool  `json:"autoacquire" description:"will automatically acquire an ip in this network if set to true, default is true"`
}

type MachineFindRequest struct {
	datastore.MachineSearchQuery
}

// MachineUpdateRequest defines the properties of a machine which can be updated.
type MachineUpdateRequest struct {
	Identifiable
	Description *string  `json:"description" description:"a description for this machine"`
	Tags        []string `json:"tags" description:"tags for this machine." optional:"true"`
}

type MachineResponse struct {
	Common
	MachineBase
	Timestamps
}

type MachineConsolePasswordRequest struct {
	ID     string `json:"id" description:"id of the machine to get the consolepassword for"`
	Reason string `json:"reason" description:"reason why the consolepassword is requested, typically a incident number with short description"`
}
type MachineConsolePasswordResponse struct {
	Common
	ConsolePassword string `json:"console_password" description:"the console password which was generated while provisioning"`
}

type MachineIPMIResponse struct {
	Common
	MachineBase
	IPMI MachineIPMI `json:"ipmi" description:"ipmi information of this machine"`
	Timestamps
}

type MachineIpmiReport struct {
	BMCIp             string
	FRU               *MachineFru
	BIOSVersion       string
	BMCVersion        string
	PowerState        string
	IndicatorLEDState string
	PowerMetric       *PowerMetric
}

type MachineIpmiReports struct {
	PartitionID string                       `json:"partitionid,omitempty" description:"the partition id for the ipmi report"`
	Reports     map[string]MachineIpmiReport `json:"reports,omitempty" description:"uuid to machinereport"`
}

type MachineIpmiReportResponse struct {
	Updated []string `json:"updated" description:"the machine uuids that triggered an update of ipmi data"`
	Created []string `json:"created" description:"the machine uuids that triggered a creation of a machine entity"`
}

type MachineReinstallRequest struct {
	Common
	ImageID string `json:"imageid" description:"the image id to be installed"`
}

type MachineAbortReinstallRequest struct {
	PrimaryDiskWiped bool `json:"primary_disk_wiped" description:"indicates whether the primary disk is already wiped"`
}

type MachineVPN struct {
	ControlPlaneAddress string `json:"address" description:"address of VPN control plane"`
	AuthKey             string `json:"auth_key" description:"auth key used to connect to VPN"`
	Connected           bool   `json:"connected" description:"connected to the VPN"`
}

func NewMetalMachineHardware(r *MachineHardware) metal.MachineHardware {
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
	for _, d := range r.Disks {
		disk := metal.BlockDevice{
			Name: d.Name,
			Size: d.Size,
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
	var powerMetric *metal.PowerMetric
	if r.PowerMetric != nil {
		powerMetric = &metal.PowerMetric{
			AverageConsumedWatts: r.PowerMetric.AverageConsumedWatts,
			IntervalInMin:        r.PowerMetric.IntervalInMin,
			MaxConsumedWatts:     r.PowerMetric.MaxConsumedWatts,
			MinConsumedWatts:     r.PowerMetric.MinConsumedWatts,
		}
	}

	return metal.IPMI{
		Address:    r.Address,
		MacAddress: r.MacAddress,
		User:       r.User,
		Password:   r.Password,
		Interface:  r.Interface,
		BMCVersion: r.BMCVersion,
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
		PowerState:  r.PowerState,
		PowerMetric: powerMetric,
	}
}

func NewMachineIPMIResponse(m *metal.Machine, s *metal.Size, p *metal.Partition, i *metal.Image, ec *metal.ProvisioningEventContainer) *MachineIPMIResponse {
	machineResponse := NewMachineResponse(m, s, p, i, ec)

	var powerMetric *PowerMetric
	if m.IPMI.PowerMetric != nil {
		powerMetric = &PowerMetric{
			AverageConsumedWatts: m.IPMI.PowerMetric.AverageConsumedWatts,
			IntervalInMin:        m.IPMI.PowerMetric.IntervalInMin,
			MaxConsumedWatts:     m.IPMI.PowerMetric.MaxConsumedWatts,
			MinConsumedWatts:     m.IPMI.PowerMetric.MinConsumedWatts,
		}
	}

	return &MachineIPMIResponse{
		Common:      machineResponse.Common,
		MachineBase: machineResponse.MachineBase,
		IPMI: MachineIPMI{
			Address:     m.IPMI.Address,
			MacAddress:  m.IPMI.MacAddress,
			User:        m.IPMI.User,
			Password:    m.IPMI.Password,
			Interface:   m.IPMI.Interface,
			BMCVersion:  m.IPMI.BMCVersion,
			PowerState:  m.IPMI.PowerState,
			PowerMetric: powerMetric,
			Fru: MachineFru{
				ChassisPartNumber:   &m.IPMI.Fru.ChassisPartNumber,
				ChassisPartSerial:   &m.IPMI.Fru.ChassisPartSerial,
				BoardMfg:            &m.IPMI.Fru.BoardMfg,
				BoardMfgSerial:      &m.IPMI.Fru.BoardMfgSerial,
				BoardPartNumber:     &m.IPMI.Fru.BoardPartNumber,
				ProductManufacturer: &m.IPMI.Fru.ProductManufacturer,
				ProductPartNumber:   &m.IPMI.Fru.ProductPartNumber,
				ProductSerial:       &m.IPMI.Fru.ProductSerial,
			},
		},
		Timestamps: machineResponse.Timestamps,
	}
}

func NewMachineResponse(m *metal.Machine, s *metal.Size, p *metal.Partition, i *metal.Image, ec *metal.ProvisioningEventContainer) *MachineResponse {
	var hardware MachineHardware

	nics := MachineNics{}
	for i := range m.Hardware.Nics {
		n := m.Hardware.Nics[i]
		neighs := MachineNics{}
		for j := range n.Neighbors {
			neigh := n.Neighbors[j]
			neighs = append(neighs, MachineNic{MacAddress: string(neigh.MacAddress), Name: neigh.Name, Neighbors: MachineNics{}})
		}
		nic := MachineNic{
			MacAddress: string(n.MacAddress),
			Name:       n.Name,
			Neighbors:  neighs,
		}
		nics = append(nics, nic)
	}

	disks := []MachineBlockDevice{}
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

	var allocation *MachineAllocation
	if m.Allocation != nil {
		var networks []MachineNetwork
		for _, nw := range m.Allocation.MachineNetworks {
			ips := append([]string{}, nw.IPs...)
			nt, err := nw.NetworkType()
			if err != nil {
				continue
			}

			network := MachineNetwork{
				NetworkID:           nw.NetworkID,
				IPs:                 ips,
				Vrf:                 nw.Vrf,
				ASN:                 int64(nw.ASN),
				Nat:                 nw.Nat,
				DestinationPrefixes: nw.DestinationPrefixes,
				Prefixes:            nw.Prefixes,
				NetworkType:         nt.Name,
				// FIXME: Both following fields are deprecated and for backward compatibility reasons only
				Private:  nt.Private,
				Underlay: nt.Underlay,
			}
			networks = append(networks, network)
		}

		allocation = &MachineAllocation{
			Creator:          m.Allocation.Creator,
			Created:          m.Allocation.Created,
			Name:             m.Allocation.Name,
			Description:      m.Allocation.Description,
			Image:            NewImageResponse(i),
			Project:          m.Allocation.Project,
			Hostname:         m.Allocation.Hostname,
			SSHPubKeys:       m.Allocation.SSHPubKeys,
			UserData:         m.Allocation.UserData,
			MachineNetworks:  networks,
			Succeeded:        m.Allocation.Succeeded,
			FilesystemLayout: NewFilesystemLayoutResponse(m.Allocation.FilesystemLayout),
			Role:             string(m.Allocation.Role),
			VPN:              NewMachineVPN(m.Allocation.VPN),
		}

		allocation.Reinstall = m.Allocation.Reinstall
		if m.Allocation.MachineSetup != nil {
			allocation.BootInfo = &BootInfo{
				ImageID:      m.Allocation.MachineSetup.ImageID,
				PrimaryDisk:  m.Allocation.MachineSetup.PrimaryDisk,
				OSPartition:  m.Allocation.MachineSetup.OSPartition,
				Initrd:       m.Allocation.MachineSetup.Initrd,
				Cmdline:      m.Allocation.MachineSetup.Cmdline,
				Kernel:       m.Allocation.MachineSetup.Kernel,
				BootloaderID: m.Allocation.MachineSetup.BootloaderID,
			}
		}
	}

	tags := []string{}
	if len(m.Tags) > 0 {
		tags = m.Tags
	}
	liveliness := ""
	if ec != nil {
		liveliness = string(ec.Liveliness)
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
			BIOS: MachineBIOS{
				Version: m.BIOS.Version,
				Vendor:  m.BIOS.Vendor,
				Date:    m.BIOS.Date,
			},
			State: MachineState{
				Value:              string(m.State.Value),
				Description:        m.State.Description,
				Issuer:             m.State.Issuer,
				MetalHammerVersion: m.State.MetalHammerVersion,
			},
			LEDState: ChassisIdentifyLEDState{
				Value:       string(m.LEDState.Value),
				Description: m.LEDState.Description,
			},
			Liveliness:               liveliness,
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
			Events:               es,
			LastEventTime:        nil,
			LastErrorEvent:       nil,
			CrashLoop:            false,
			FailedMachineReclaim: false,
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
	var lastErrorEvent *MachineProvisioningEvent
	if ec.LastErrorEvent != nil {
		lastErrorEvent = &MachineProvisioningEvent{
			Time:    ec.LastErrorEvent.Time,
			Event:   ec.LastErrorEvent.Event.String(),
			Message: ec.LastErrorEvent.Message,
		}
	}
	return &MachineRecentProvisioningEvents{
		Events:               es,
		LastEventTime:        ec.LastEventTime,
		LastErrorEvent:       lastErrorEvent,
		CrashLoop:            ec.CrashLoop,
		FailedMachineReclaim: ec.FailedMachineReclaim,
	}
}

func NewMachineVPN(m *metal.MachineVPN) *MachineVPN {
	if m == nil {
		return nil
	}

	return &MachineVPN{
		ControlPlaneAddress: m.ControlPlaneAddress,
		AuthKey:             m.AuthKey,
		Connected:           m.Connected,
	}
}
