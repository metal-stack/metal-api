package metal

import (
	"fmt"
	"time"

	"github.com/dustin/go-humanize"
)

// A MState is an enum which indicates the state of a machine
type MState string

// The enums for the machine states.
const (
	AvailableState MState = ""
	ReservedState  MState = "RESERVED"
	LockedState    MState = "LOCKED"
)

var (
	// AllStates contains all possible values of a machine state
	AllStates = []MState{AvailableState, ReservedState, LockedState}
)

// A MachineState describes the state of a machine. If the Value is AvailableState,
// the machine will be available for allocation. In all other cases the allocation
// must explicitly point to this machine.
type MachineState struct {
	Value       MState `rethinkdb:"value" json:"value"`
	Description string `rethinkdb:"description" json:"description"`
}

// MachineStateFrom converts a machineState string to the type
func MachineStateFrom(name string) (MState, error) {
	switch name {
	case string(AvailableState):
		return AvailableState, nil
	case string(ReservedState):
		return ReservedState, nil
	case string(LockedState):
		return LockedState, nil
	default:
		return "", fmt.Errorf("unknown MachineState:%s", name)
	}
}

// LEDState is the state of the LED of the Machine
type LEDState string

const (
	// LEDStateOn LED is on
	LEDStateOn LEDState = "LED-ON"
	// LEDStateOff LED is off
	LEDStateOff LEDState = "LED-OFF"
)

// LEDStateFrom converts an LEDState string to the corresponding type
func LEDStateFrom(name string) (LEDState, error) {
	switch name {
	case string(LEDStateOff):
		return LEDStateOff, nil
	case string(LEDStateOn):
		return LEDStateOn, nil
	default:
		return "", fmt.Errorf("unknown LEDState:%s", name)
	}
}

// A ChassisIdentifyLEDState describes the state of a chassis identify LED, i.e. LED-ON/LED-OFF.
type ChassisIdentifyLEDState struct {
	Value       LEDState `rethinkdb:"value" json:"value"`
	Description string   `rethinkdb:"description" json:"description"`
}

// A Machine is a piece of metal which is under the control of our system. It registers itself
// and can be allocated or freed. If the machine is allocated, the substructure Allocation will
// be filled. Any unallocated (free) machine won't have such values.
type Machine struct {
	Base
	Allocation  *MachineAllocation      `rethinkdb:"allocation" json:"allocation"`
	PartitionID string                  `rethinkdb:"partitionid" json:"partitionid"`
	SizeID      string                  `rethinkdb:"sizeid" json:"sizeid"`
	RackID      string                  `rethinkdb:"rackid" json:"rackid"`
	Hardware    MachineHardware         `rethinkdb:"hardware" json:"hardware"`
	State       MachineState            `rethinkdb:"state" json:"state"`
	LEDState    ChassisIdentifyLEDState `rethinkdb:"ledstate" json:"ledstate"`
	Tags        []string                `rethinkdb:"tags" json:"tags"`
	IPMI        IPMI                    `rethinkdb:"ipmi" json:"ipmi"`
	BIOS        BIOS                    `rethinkdb:"bios" json:"bios"`
}

// Machines is a slice of Machine
type Machines []Machine

// IsFirewall returns true if this machine is a firewall machine.
func (m *Machine) IsFirewall(iMap ImageMap) bool {
	if m.Allocation == nil {
		return false
	}
	image, ok := iMap[m.Allocation.ImageID]
	if !ok {
		return false
	}
	if !image.HasFeature(ImageFeatureFirewall) {
		return false
	}
	if len(m.Allocation.MachineNetworks) <= 1 {
		return false
	}
	return true
}

// A MachineAllocation stores the data which are only present for allocated machines.
type MachineAllocation struct {
	Created         time.Time         `rethinkdb:"created" json:"created"`
	Name            string            `rethinkdb:"name" json:"name"`
	Description     string            `rethinkdb:"description" json:"description"`
	Project         string            `rethinkdb:"project" json:"project"`
	ImageID         string            `rethinkdb:"imageid" json:"imageid"`
	MachineNetworks []*MachineNetwork `rethinkdb:"networks" json:"networks"`
	Hostname        string            `rethinkdb:"hostname" json:"hostname"`
	SSHPubKeys      []string          `rethinkdb:"sshPubKeys" json:"sshPubKeys"`
	UserData        string            `rethinkdb:"userdata" json:"userdata"`
	ConsolePassword string            `rethinkdb:"console_password" json:"console_password"`
	Succeeded       bool              `rethinkdb:"succeeded" json:"succeeded"`
	Reinstall       bool              `rethinkdb:"reinstall" json:"reinstall"`
	PrimaryDisk     string            `rethinkdb:"primarydisk" json:"primarydisk"`
	OSPartition     string            `rethinkdb:"ospartition" json:"ospartition"`
	Initrd          string            `rethinkdb:"initrd" json:"initrd"`
	Cmdline         string            `rethinkdb:"cmdline" json:"cmdline"`
	Kernel          string            `rethinkdb:"kernel" json:"kernel"`
	BootloaderID    string            `rethinkdb:"bootloaderid" json:"bootloaderid"`
}

// ByProjectID creates a map of machines with the project id as the index.
func (ms Machines) ByProjectID() map[string]Machines {
	res := make(map[string]Machines)
	for i, m := range ms {
		if m.Allocation != nil {
			res[m.Allocation.Project] = append(res[m.Allocation.Project], ms[i])
		}
	}
	return res
}

// MachineNetwork stores the Network details of the machine
type MachineNetwork struct {
	NetworkID           string   `rethinkdb:"networkid" json:"networkid"`
	Prefixes            []string `rethinkdb:"prefixes" json:"prefixes"`
	IPs                 []string `rethinkdb:"ips" json:"ips"`
	DestinationPrefixes []string `rethinkdb:"destinationprefixes" json:"destinationprefixes"`
	Vrf                 uint     `rethinkdb:"vrf" json:"vrf"`
	Private             bool     `rethinkdb:"private" json:"private"`
	ASN                 int64    `rethinkdb:"asn" json:"asn"`
	Nat                 bool     `rethinkdb:"nat" json:"nat"`
	Underlay            bool     `rethinkdb:"underlay" json:"underlay"`
}

// MachineHardware stores the data which is collected by our system on the hardware when it registers itself.
type MachineHardware struct {
	Memory   uint64        `rethinkdb:"memory" json:"memory"`
	CPUCores int           `rethinkdb:"cpu_cores" json:"cpu_cores"`
	Nics     Nics          `rethinkdb:"network_interfaces" json:"network_interfaces"`
	Disks    []BlockDevice `rethinkdb:"block_devices" json:"block_devices"`
}

// MachineLiveliness indicates the liveliness of a machine
type MachineLiveliness string

// The enums for the machine liveliness states.
const (
	MachineLivelinessAlive   MachineLiveliness = "Alive"
	MachineLivelinessDead    MachineLiveliness = "Dead"
	MachineLivelinessUnknown MachineLiveliness = "Unknown"
	MachineDeadAfter         time.Duration     = 5 * time.Minute
	MachineResurrectAfter    time.Duration     = time.Hour
)

// Is return true if given liveliness is equal to specific Liveliness
func (l MachineLiveliness) Is(liveliness string) bool {
	return string(l) == liveliness
}

// DiskCapacity calculates the capacity of all disks.
func (hw *MachineHardware) DiskCapacity() uint64 {
	var c uint64
	for _, d := range hw.Disks {
		c += d.Size
	}
	return c
}

// ReadableSpec returns a human readable string for the hardware.
func (hw *MachineHardware) ReadableSpec() string {
	return fmt.Sprintf("Cores: %d, Memory: %s, Storage: %s", hw.CPUCores, humanize.Bytes(hw.Memory), humanize.Bytes(hw.DiskCapacity()))
}

// BlockDevice information.
type BlockDevice struct {
	Name       string           `rethinkdb:"name" json:"name"`
	Size       uint64           `rethinkdb:"size" json:"size"`
	Partitions []*DiskPartition `rethinkdb:"partitions" json:"partitions"`
	Primary    bool             `rethinkdb:"primary" json:"primary"`
}

// DiskPartition defines a disk partition
type DiskPartition struct {
	Label        string            `rethinkdb:"label" json:"label"`
	Device       string            `rethinkdb:"device" json:"device"`
	Number       uint              `rethinkdb:"number" json:"number"`
	MountPoint   string            `rethinkdb:"mountpoint" json:"mountpoint"`
	MountOptions []string          `rethinkdb:"mountoptions" json:"mountoptions"`
	Size         int64             `rethinkdb:"size" json:"size"`
	Filesystem   string            `rethinkdb:"filesystem" json:"filesystem"`
	GPTType      string            `rethinkdb:"gpttyoe" json:"gpttyoe"`
	GPTGuid      string            `rethinkdb:"gptguid" json:"gptguid"`
	Properties   map[string]string `rethinkdb:"properties" json:"properties"`
	ContainsOS   bool              `rethinkdb:"containsos" json:"containsos"`
}

// Fru (Field Replaceable Unit) data
type Fru struct {
	ChassisPartNumber   string `rethinkdb:"chassis_part_number" json:"chassis_part_number"`
	ChassisPartSerial   string `rethinkdb:"chassis_part_serial" json:"chassis_part_serial"`
	BoardMfg            string `rethinkdb:"board_mfg" json:"board_mfg"`
	BoardMfgSerial      string `rethinkdb:"board_mfg_serial" json:"board_mfg_serial"`
	BoardPartNumber     string `rethinkdb:"board_part_number" json:"board_part_number"`
	ProductManufacturer string `rethinkdb:"product_manufacturer" json:"product_manufacturer"`
	ProductPartNumber   string `rethinkdb:"product_part_number" json:"product_part_number"`
	ProductSerial       string `rethinkdb:"product_serial" json:"product_serial"`
}

// IPMI connection data
type IPMI struct {
	// Address is host:port of the connection to the ipmi BMC, host can be either a ip address or a hostname
	Address    string `rethinkdb:"address" json:"address"`
	MacAddress string `rethinkdb:"mac" json:"mac"`
	User       string `rethinkdb:"user" json:"user"`
	Password   string `rethinkdb:"password" json:"password"`
	Interface  string `rethinkdb:"interface" json:"interface"`
	Fru        Fru    `rethinkdb:"fru" json:"fru"`
	BMCVersion string `rethinkdb:"bmcversion" json:"bmcversion"`
}

// BIOS contains machine bios information
type BIOS struct {
	Version string `rethinkdb:"version" json:"version"`
	Vendor  string `rethinkdb:"vendor" json:"vendor"`
	Date    string `rethinkdb:"date" json:"date"`
}

// HasMAC returns true if this machine has the given MAC.
func (m *Machine) HasMAC(mac string) bool {
	for _, nic := range m.Hardware.Nics {
		if string(nic.MacAddress) == mac {
			return true
		}
	}
	return false
}

// A MachineCommand is an alias of a string
type MachineCommand string

// our supported machines commands.
const (
	MachineOnCmd             MachineCommand = "ON"
	MachineOffCmd            MachineCommand = "OFF"
	MachineResetCmd          MachineCommand = "RESET"
	MachineBiosCmd           MachineCommand = "BIOS"
	MachineReinstall         MachineCommand = "REINSTALL"
	MachineAbortReinstall    MachineCommand = "ABORT-REINSTALL"
	ChassisIdentifyLEDOnCmd  MachineCommand = "LED-ON"
	ChassisIdentifyLEDOffCmd MachineCommand = "LED-OFF"
)

// A MachineExecCommand can be sent via a MachineEvent to execute
// the command against the specific machine. The specified command
// should be executed against the given target machine. The parameters
// is an optional array of strings which are implementation specific
// and dependent of the command.
type MachineExecCommand struct {
	Target  *Machine       `json:"target,omitempty"`
	Command MachineCommand `json:"cmd,omitempty"`
	Params  []string       `json:"params,omitempty"`
}

// MachineEvent is propagated when a machine is create/updated/deleted.
type MachineEvent struct {
	Type EventType           `json:"type,omitempty"`
	Old  *Machine            `json:"old,omitempty"`
	New  *Machine            `json:"new,omitempty"`
	Cmd  *MachineExecCommand `json:"cmd,omitempty"`
}
