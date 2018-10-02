package metal

import "time"

type Device struct {
	ID           string    `json:"id" description:"a unique ID" unique:"true" readOnly:"true" modelDescription:"A device representing a bare metal machine." rethinkdb:"id,omitempty"`
	Name         string    `json:"name" description:"the name of the device" rethinkdb:"name"`
	Description  string    `json:"description,omitempty" description:"a description for this machine" optional:"true" rethinkdb:"description"`
	Created      time.Time `json:"created" description:"the creation time of this machine" optional:"true" readOnly:"true" rethinkdb:"created"`
	Changed      time.Time `json:"changed" description:"the last changed timestamp" optional:"true" readOnly:"true" rethinkdb:"changed"`
	Project      string    `json:"project" description:"the project that this device is assigned to" rethinkdb:"project"`
	Facility     Facility  `json:"facility" description:"the facility assigned to this device" readOnly:"true" rethinkdb:"-"`
	Image        *Image    `json:"image" description:"the image assigned to this device" readOnly:"true"  rethinkdb:"-"`
	Size         Size      `json:"size" description:"the size of this device" readOnly:"true" rethinkdb:"-"`
	FacilityID   string    `json:"-" rethinkdb:"facilityid"`
	ImageID      string    `json:"-" rethinkdb:"imageid"`
	SizeID       string    `json:"-" rethinkdb:"sizeid"`
	MACAddresses []string  `json:"macAddresses" description:"the list of mac addresses in this device" readOnly:"true" rethinkdb:"macAddresses"`
}

//`rethinkdb:"author_ids,reference" rethinkdb_ref:"id"`

// HasMAC returns true if this device has the given MAC.
func (d *Device) HasMAC(m string) bool {
	for _, mac := range d.MACAddresses {
		if mac == m {
			return true
		}
	}
	return false
}
