package metal

import "time"

type Device struct {
	ID          string         `json:"id" description:"a unique ID" unique:"true" readOnly:"true" modelDescription:"A device representing a bare metal machine." rethinkdb:"id,omitempty"`
	Name        string         `json:"name" description:"the name of the device" rethinkdb:"name"`
	Description string         `json:"description,omitempty" description:"a description for this machine" optional:"true" rethinkdb:"description"`
	Created     time.Time      `json:"created" description:"the creation time of this machine" optional:"true" readOnly:"true" rethinkdb:"created"`
	Changed     time.Time      `json:"changed" description:"the last changed timestamp" optional:"true" readOnly:"true" rethinkdb:"changed"`
	LastPing    time.Time      `json:"last_ping" description:"the timestamp of the last phone home call/ping from the device" optional:"true" readOnly:"true" rethinkdb:"last_ping"`
	Project     string         `json:"project" description:"the project that this device is assigned to" rethinkdb:"project"`
	Site        Site           `json:"site" description:"the site assigned to this device" readOnly:"true" rethinkdb:"-"`
	SiteID      string         `json:"-" rethinkdb:"siteid"`
	Image       *Image         `json:"image" description:"the image assigned to this device" readOnly:"true"  rethinkdb:"-"`
	ImageID     string         `json:"-" rethinkdb:"imageid"`
	Size        *Size          `json:"size" description:"the size of this device" readOnly:"true" rethinkdb:"-"`
	SizeID      string         `json:"-" rethinkdb:"sizeid"`
	Hardware    DeviceHardware `json:"hardware" description:"the hardware of this device" rethinkdb:"hardware"`
	Cidr        string         `json:"cidr" description:"the cidr address of the allocated device" rethinkdb:"cidr"`
	Hostname    string         `json:"hostname" description:"the hostname which will be used when creating the device" rethinkdb:"-"`
	SSHPubKeys  []string       `json:"ssh_pub_keys" description:"the public ssh keys to access the device with" rethinkdb:"sshPubKeys"`
}

type DeviceHardware struct {
	Memory   uint64        `json:"memory" description:"the total memory of the device" rethinkdb:"memory"`
	CPUCores int           `json:"cpu_cores" description:"the number of cpu cores" rethinkdb:"cpu_cores"`
	Nics     []Nic         `json:"nics" description:"the list of network interfaces of this device" rethinkdb:"network_interfaces"`
	Disks    []BlockDevice `json:"disks" description:"the list of block devices of this device" rethinkdb:"block_devices"`
}

type Nic struct {
	MacAddress string `json:"mac"  description:"the mac address of this network interface" rethinkdb:"macAddress"`
	Name       string `json:"name"  description:"the name of this network interface" rethinkdb:"name"`
}

type BlockDevice struct {
	Name string `json:"name" description:"the name of this block device" rethinkdb:"name"`
	Size uint64 `json:"size" description:"the size of this block device" rethinkdb:"size"`
}

type IPMI struct {
	ID         string `json:"-" rethinkdb:"id"`
	Address    string `json:"address" rethinkdb:"address" modelDescription:"The IPMI connection data"`
	MacAddress string `json:"mac" rethinkdb:"mac"`
	User       string `json:"user" rethinkdb:"user"`
	Password   string `json:"password" rethinkdb:"password"`
	Interface  string `json:"interface" rethinkdb:"interface"`
}

//`rethinkdb:"author_ids,reference" rethinkdb_ref:"id"`

// HasMAC returns true if this device has the given MAC.
func (d *Device) HasMAC(m string) bool {
	for _, nic := range d.Hardware.Nics {
		if nic.MacAddress == m {
			return true
		}
	}
	return false
}

type DeviceEvent struct {
	Type EventType `json:"type,omitempty"`
	Old  *Device   `json:"old,omitempty"`
	New  *Device   `json:"new,omitempty"`
}

type DeviceWithPhoneHomeToken struct {
	Device         *Device `json:"device"`
	PhoneHomeToken string  `json:"phone_home_token"`
}
