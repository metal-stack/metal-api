package metal

import "time"

type Device struct {
	ID          string         `json:"id" description:"a unique ID" unique:"true" readOnly:"true" modelDescription:"A device representing a bare metal machine." rethinkdb:"id,omitempty"`
	Name        string         `json:"name" description:"the name of the device" rethinkdb:"name"`
	Description string         `json:"description,omitempty" description:"a description for this machine" optional:"true" rethinkdb:"description"`
	Created     time.Time      `json:"created" description:"the creation time of this machine" optional:"true" readOnly:"true" rethinkdb:"created"`
	Changed     time.Time      `json:"changed" description:"the last changed timestamp" optional:"true" readOnly:"true" rethinkdb:"changed"`
	Project     string         `json:"project" description:"the project that this device is assigned to" rethinkdb:"project"`
	Site        Facility       `json:"site" description:"the site assigned to this device" readOnly:"true" rethinkdb:"-"`
	SiteID      string         `json:"-" rethinkdb:"siteid"`
	Image       *Image         `json:"image" description:"the image assigned to this device" readOnly:"true"  rethinkdb:"-"`
	ImageID     string         `json:"-" rethinkdb:"imageid"`
	Size        *Size          `json:"size" description:"the size of this device" readOnly:"true" rethinkdb:"-"`
	SizeID      string         `json:"-" rethinkdb:"sizeid"`
	Hardware    DeviceHardware `json:"hardware" description:"the hardware of this device" rethinkdb:"hardware"`
	Cidr        string         `json:"cidr" description:"the cidr address of the allocated device" rethinkdb:"cidr"`
	Hostname    string         `json:"hostname" description:"the hostname of the device" rethinkdb:"hostname"`
	SSHPubKey   string         `json:"ssh_pub_key" description:"the public ssh key to access the device with" rethinkdb:"sshPubKey"`
}

type DeviceHardware struct {
	Memory   int64         `json:"memory" description:"the total memory of the device" rethinkdb:"memory"`
	CPUCores uint32        `json:"cpu_cores" description:"the total memory of the device" rethinkdb:"cpu_cores"`
	Nics     []Nic         `json:"nics" description:"the list of network interfaces of this device" rethinkdb:"network_interfaces"`
	Disks    []BlockDevice `json:"disks" description:"the list of block devices of this device" rethinkdb:"block_devices"`
}

type Nic struct {
	MacAddress string   `json:"mac"  description:"the mac address of this network interface" rethinkdb:"macAddress"`
	Name       string   `json:"name"  description:"the name of this network interface" rethinkdb:"name"`
	Vendor     string   `json:"vendor"  description:"the vendor of this network interface" rethinkdb:"vendor"`
	Features   []string `json:"features"  description:"the features of this network interface" rethinkdb:"features"`
}

type BlockDevice struct {
	Name string `json:"name" description:"the name of this block device" rethinkdb:"name"`
	Size uint64 `json:"size" description:"the size of this block device" rethinkdb:"size"`
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
