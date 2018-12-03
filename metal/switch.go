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
	RackID      string       `json:"rack_id" description:"the id of the rack in which this switch is located" rethinkdb:"rackid"`
	Created     time.Time    `json:"created" description:"the creation time of this switch" optional:"true" readOnly:"true" rethinkdb:"created"`
	Changed     time.Time    `json:"changed" description:"the last changed timestamp" optional:"true" readOnly:"true" rethinkdb:"changed"`
}

// Connection between switch port and device.
type Connection struct {
	Nic      Nic    `json:"nic" description:"a network interface on the switch" rethinkdb:"nic"`
	DeviceID string `json:"device_id,omitempty" optional:"true" description:"the device id of the device connected to the nic" rethinkdb:"deviceid"`
}

// ConnectDevice iterates over all switch nics and device nic neighbor
// to find existing wire connections.
// Implementation is very inefficient, will also find all connections,
// which should not happen in a real environment.
func (s *Switch) ConnectDevice(device *Device) {
	// first remove all existing connections to this device.
	newConnections := make([]Connection, 0)
	for i, switchConn := range s.Connections {
		if switchConn.DeviceID == device.ID {
			// stolen from: https://github.com/golang/go/wiki/SliceTricks#delete
			newConnections = append(newConnections[:i], newConnections[i+1:]...)
		}
	}
	s.Connections = newConnections

	// calculate the connections for this device
	for _, switchNic := range s.Nics {
		for _, deviceNic := range device.Hardware.Nics {
			for _, neigh := range deviceNic.Neighbors {
				if switchNic.MacAddress == neigh.MacAddress {
					conn := Connection{
						Nic:      switchNic,
						DeviceID: device.ID,
					}
					s.Connections = append(s.Connections, conn)
				}
			}
		}
	}

}
