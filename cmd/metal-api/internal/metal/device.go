package metal

import "time"

// A Device is a piece of metal which is under the control of our system. It registers itself
// and can be allocated or freed. If the device is allocated, the substructure Allocation will
// be filled. Any unallocated (free) device won't have such values.
type Device struct {
	Base
	Partition   Partition         `json:"partition" modelDescription:"A device representing a bare metal machine." description:"the partition assigned to this device" readOnly:"true" rethinkdb:"-"`
	PartitionID string            `json:"-" rethinkdb:"partitionid"`
	RackID      string            `json:"rackid" description:"the rack assigned to this device" readOnly:"true" rethinkdb:"rackid"`
	Size        *Size             `json:"size" description:"the size of this device" readOnly:"true" rethinkdb:"-"`
	SizeID      string            `json:"-" rethinkdb:"sizeid"`
	Hardware    DeviceHardware    `json:"hardware" description:"the hardware of this device" rethinkdb:"hardware"`
	Allocation  *DeviceAllocation `json:"allocation" description:"the allocation data of an allocated device" rethinkdb:"allocation"`
}

// A DeviceAllocation stores the data which are only present for allocated devices.
type DeviceAllocation struct {
	Created         time.Time `json:"created" description:"the time when the device was created" rethinkdb:"created"`
	Name            string    `json:"name" description:"the name of the device" rethinkdb:"name"`
	Description     string    `json:"description,omitempty" description:"a description for this machine" optional:"true" rethinkdb:"description"`
	LastPing        time.Time `json:"last_ping" description:"the timestamp of the last phone home call/ping from the device" optional:"true" readOnly:"true" rethinkdb:"last_ping"`
	Tenant          string    `json:"tenant" description:"the tenant that this device is assigned to" rethinkdb:"tenant"`
	Project         string    `json:"project" description:"the project that this device is assigned to" rethinkdb:"project"`
	Image           *Image    `json:"image" description:"the image assigned to this device" readOnly:"true" optional:"true" rethinkdb:"-"`
	ImageID         string    `json:"-" rethinkdb:"imageid"`
	Cidr            string    `json:"cidr" description:"the cidr address of the allocated device" rethinkdb:"cidr"`
	Vrf             uint      `json:"vrf" description:"the vrf of the allocated device" rethinkdb:"vrf"`
	Hostname        string    `json:"hostname" description:"the hostname which will be used when creating the device" rethinkdb:"hostname"`
	SSHPubKeys      []string  `json:"ssh_pub_keys" description:"the public ssh keys to access the device with" rethinkdb:"sshPubKeys"`
	UserData        string    `json:"user_data,omitempty" description:"userdata to execute post installation tasks" optional:"true" rethinkdb:"userdata"`
	ConsolePassword string    `json:"console_password" description:"the console password which was generated while provisioning" optional:"true" rethinkdb:"console_password"`
}

// DeviceHardware stores the data which is collected by our system on the hardware when it registers itself.
type DeviceHardware struct {
	Memory   uint64        `json:"memory" description:"the total memory of the device" rethinkdb:"memory"`
	CPUCores int           `json:"cpu_cores" description:"the number of cpu cores" rethinkdb:"cpu_cores"`
	Nics     Nics          `json:"nics" description:"the list of network interfaces of this device" rethinkdb:"network_interfaces"`
	Disks    []BlockDevice `json:"disks" description:"the list of block devices of this device" rethinkdb:"block_devices"`
}

// BlockDevice information.
type BlockDevice struct {
	Name string `json:"name" description:"the name of this block device" rethinkdb:"name"`
	Size uint64 `json:"size" description:"the size of this block device" rethinkdb:"size"`
}

// IPMI connection data
type IPMI struct {
	ID         string `json:"-" rethinkdb:"id"`
	Address    string `json:"address" rethinkdb:"address" modelDescription:"The IPMI connection data"`
	MacAddress string `json:"mac" rethinkdb:"mac"`
	User       string `json:"user" rethinkdb:"user"`
	Password   string `json:"password" rethinkdb:"password"`
	Interface  string `json:"interface" rethinkdb:"interface"`
}

// HasMAC returns true if this device has the given MAC.
func (d *Device) HasMAC(mac string) bool {
	for _, nic := range d.Hardware.Nics {
		if string(nic.MacAddress) == mac {
			return true
		}
	}
	return false
}

// DeviceEvent is propagated when a device is create/updated/deleted.
type DeviceEvent struct {
	Type EventType `json:"type,omitempty"`
	Old  *Device   `json:"old,omitempty"`
	New  *Device   `json:"new,omitempty"`
}

// DeviceWithPhoneHomeToken enriches a device with a token. This is only
// used for the communication with the client.
type DeviceWithPhoneHomeToken struct {
	Device         *Device `json:"device"`
	PhoneHomeToken string  `json:"phone_home_token"`
}
