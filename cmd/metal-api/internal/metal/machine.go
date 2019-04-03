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
)

var (
	// AllStates contains all possible values of a machine state
	AllStates = []MState{AvailableState, ReservedState}
)

// A MachineState describes the state of a machine. If the Value is AvailableState,
// the machine will be available for allocation. In all other cases the allocation
// must explicitly point to this machine.
type MachineState struct {
	Value       MState `json:"value" rethinkdb:"value" description:"the state of this machine. empty means available for all"`
	Description string `json:"description" rethinkdb:"description" description:"a description why this machine is in the given state"`
}

// A Machine is a piece of metal which is under the control of our system. It registers itself
// and can be allocated or freed. If the machine is allocated, the substructure Allocation will
// be filled. Any unallocated (free) machine won't have such values.
type Machine struct {
	Base
	Partition   Partition          `json:"partition" modelDescription:"A machine representing a bare metal machine." description:"the partition assigned to this machine" readOnly:"true" rethinkdb:"-"`
	PartitionID string             `json:"-" rethinkdb:"partitionid"`
	RackID      string             `json:"rackid" description:"the rack assigned to this machine" readOnly:"true" rethinkdb:"rackid"`
	Size        *Size              `json:"size" description:"the size of this machine" readOnly:"true" rethinkdb:"-"`
	SizeID      string             `json:"-" rethinkdb:"sizeid"`
	Hardware    MachineHardware    `json:"hardware" description:"the hardware of this machine" rethinkdb:"hardware"`
	Allocation  *MachineAllocation `json:"allocation" description:"the allocation data of an allocated machine" rethinkdb:"allocation"`
	Tags        []string           `json:"tags" description:"tags for this machine" rethinkdb:"tags"`
	State       MachineState       `json:"state" rethinkdb:"state" description:"the state of this machine"`
}

// A MachineAllocation stores the data which are only present for allocated machines.
type MachineAllocation struct {
	Created         time.Time `json:"created" description:"the time when the machine was created" rethinkdb:"created"`
	Name            string    `json:"name" description:"the name of the machine" rethinkdb:"name"`
	Description     string    `json:"description,omitempty" description:"a description for this machine" optional:"true" rethinkdb:"description"`
	LastPing        time.Time `json:"last_ping" description:"the timestamp of the last phone home call/ping from the machine" optional:"true" readOnly:"true" rethinkdb:"last_ping"`
	Tenant          string    `json:"tenant" description:"the tenant that this machine is assigned to" rethinkdb:"tenant"`
	Project         string    `json:"project" description:"the project that this machine is assigned to" rethinkdb:"project"`
	Image           *Image    `json:"image" description:"the image assigned to this machine" readOnly:"true" optional:"true" rethinkdb:"-"`
	ImageID         string    `json:"-" rethinkdb:"imageid"`
	Cidr            string    `json:"cidr" description:"the cidr address of the allocated machine" rethinkdb:"cidr"`
	Vrf             uint      `json:"vrf" description:"the vrf of the allocated machine" rethinkdb:"vrf"`
	Hostname        string    `json:"hostname" description:"the hostname which will be used when creating the machine" rethinkdb:"hostname"`
	SSHPubKeys      []string  `json:"ssh_pub_keys" description:"the public ssh keys to access the machine with" rethinkdb:"sshPubKeys"`
	UserData        string    `json:"user_data,omitempty" description:"userdata to execute post installation tasks" optional:"true" rethinkdb:"userdata"`
	ConsolePassword string    `json:"console_password" description:"the console password which was generated while provisioning" optional:"true" rethinkdb:"console_password"`
}

// MachineHardware stores the data which is collected by our system on the hardware when it registers itself.
type MachineHardware struct {
	Memory   uint64        `json:"memory" description:"the total memory of the machine" rethinkdb:"memory"`
	CPUCores int           `json:"cpu_cores" description:"the number of cpu cores" rethinkdb:"cpu_cores"`
	Nics     Nics          `json:"nics" description:"the list of network interfaces of this machine" rethinkdb:"network_interfaces"`
	Disks    []BlockDevice `json:"disks" description:"the list of block devices of this machine" rethinkdb:"block_devices"`
}

// ProvisioningState indicates the state of the machine during the provisioning sequence
type ProvisioningState string

var (
	// AllProvisioningStates are all provisioning states that exist
	AllProvisioningStates = map[ProvisioningState]bool{
		ProvisioningStateAlive:                true,
		ProvisioningStatePreparing:            true,
		ProvisioningStateRegistering:          true,
		ProvisioningStateWaiting:              true,
		ProvisioningStateInstalling:           true,
		ProvisioningStateInstallationFinished: true,
		ProvisioningStateProvisioned:          true,
		ProvisioningStateDead:                 true,
	}
)

// The enums for the machine provisioning states.
const (
	ProvisioningStateAlive                ProvisioningState = "Alive"
	ProvisioningStatePreparing            ProvisioningState = "Preparing"
	ProvisioningStateRegistering          ProvisioningState = "Registering"
	ProvisioningStateWaiting              ProvisioningState = "Waiting"
	ProvisioningStateInstalling           ProvisioningState = "Installing"
	ProvisioningStateInstallationFinished ProvisioningState = "InstallationFinished"
	ProvisioningStateProvisioned          ProvisioningState = "Provisioned"
	ProvisioningStateDead                 ProvisioningState = "Dead"
)

const MachineProvisioningStateHistoryLength = 10

type MachineProvisioningStateHistory []MachineProvisioningStateHistoryEntry

type MachineProvisioningStateHistoryEntry struct {
	Changed time.Time         `json:"changed" description:"the last changed timestamp" optional:"true" readOnly:"true" rethinkdb:"changed"`
	State   ProvisioningState `json:"state" description:"the state of the machine" rethinkdb:"state"`
	Message string            `json:"message" description:"the state of the machine" rethinkdb:"message"`
}

// MachineProvisioningState stores the provisioning state of the machine
type MachineProvisioningState struct {
	Changed time.Time                       `json:"changed" description:"the last changed timestamp" optional:"true" readOnly:"true" rethinkdb:"changed"`
	ID      string                          `json:"-" description:"references the machine" rethinkdb:"id"`
	State   ProvisioningState               `json:"state" description:"the state of the machine" rethinkdb:"state"`
	Message string                          `json:"message" description:"the state of the machine" rethinkdb:"message"`
	History MachineProvisioningStateHistory `json:"-" description:"the history of the last states" rethinkdb:"history"`
}

// DiskCapacity calculates the capacity of all disks.
func (hw *MachineHardware) DiskCapacity() uint64 {
	var cap uint64
	for _, d := range hw.Disks {
		cap += d.Size
	}
	return cap
}

// ReadableSpec returns a human readable string for the hardware.
func (hw *MachineHardware) ReadableSpec() string {
	return fmt.Sprintf("Cores:%d, Memory:%s, Storage: %s", hw.CPUCores, humanize.Bytes(hw.Memory), humanize.Bytes(hw.DiskCapacity()))
}

// BlockDevice information.
type BlockDevice struct {
	Name string `json:"name" description:"the name of this block device" rethinkdb:"name"`
	Size uint64 `json:"size" description:"the size of this block device" rethinkdb:"size"`
}

// Fru (Field Replaceable Unit) data
type Fru struct {
	ChassisPartNumber   string `json:"chassis_part_number,omitempty" description:"the chassis part number" rethinkdb:"chassis_part_number"`
	ChassisPartSerial   string `json:"chassis_part_serial,omitempty" description:"the chassis part serial" rethinkdb:"chassis_part_serial"`
	BoardMfg            string `json:"board_mfg,omitempty" description:"the board mfg" rethinkdb:"board_mfg"`
	BoardMfgSerial      string `json:"board_mfg_serial,omitempty" description:"the board mfg serial" rethinkdb:"board_mfg_serial"`
	BoardPartNumber     string `json:"board_part_number,omitempty" description:"the board part number" rethinkdb:"board_part_number"`
	ProductManufacturer string `json:"product_manufacturer,omitempty" description:"the product manufacturer" rethinkdb:"product_manufacturer"`
	ProductPartNumber   string `json:"product_part_number,omitempty" description:"the product part number" rethinkdb:"product_part_number"`
	ProductSerial       string `json:"product_serial,omitempty" description:"the product serial" rethinkdb:"product_serial"`
}

// IPMI connection data
type IPMI struct {
	ID         string `json:"-" rethinkdb:"id"`
	Address    string `json:"address" rethinkdb:"address" modelDescription:"The IPMI connection data"`
	MacAddress string `json:"mac" rethinkdb:"mac"`
	User       string `json:"user" rethinkdb:"user"`
	Password   string `json:"password" rethinkdb:"password"`
	Interface  string `json:"interface" rethinkdb:"interface"`
	Fru        Fru    `json:"fru" rethinkdb:"fru" modelDescription:"The Field Replaceable Unit data"`
}

// HasMAC returns true if this machine has the given MAC.
func (d *Machine) HasMAC(mac string) bool {
	for _, nic := range d.Hardware.Nics {
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
	MachineOnCmd    MachineCommand = "ON"
	MachineOffCmd   MachineCommand = "OFF"
	MachineResetCmd MachineCommand = "RESET"
	MachineBiosCmd  MachineCommand = "BIOS"
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

// MachineWithPhoneHomeToken enriches a machine with a token. This is only
// used for the communication with the client.
type MachineWithPhoneHomeToken struct {
	Machine        *Machine `json:"machine"`
	PhoneHomeToken string   `json:"phone_home_token"`
}
