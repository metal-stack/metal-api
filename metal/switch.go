package metal

import (
	"time"
)

// Switch have to register at the api. As metal-core is a stateless application running on a switch,
// the api needs persist all the information such that the core can create or restore a its entire
// switch configuration.
type Switch struct {
	ID          string       `json:"id" description:"a unique ID" unique:"true" modelDescription:"A switch that can register at the api." rethinkdb:"id,omitempty"`
	Nics        []Nic        `json:"nics" description:"the list of network interfaces on the switch" rethinkdb:"network_interfaces"`
	Connections []Connection `json:"connections" description:"a connection between a switch port and a device" rethinkdb:"connections"`
	SiteID      string       `json:"site_id" description:"the id of the site in which this switch is located" rethinkdb:"siteid"`
	Created     time.Time    `json:"created" description:"the creation time of this switch" optional:"true" readOnly:"true" rethinkdb:"created"`
	Changed     time.Time    `json:"changed" description:"the last changed timestamp" optional:"true" readOnly:"true" rethinkdb:"changed"`
}

// Connection between switch port and device.
type Connection struct {
	Nic      Nic    `json:"nic" description:"a network interface on the switch" rethinkdb:"nic"`
	DeviceID string `json:"device_id,omitempty" optional:"true" description:"the device id of the device connected to the nic" rethinkdb:"deviceid"`
}
