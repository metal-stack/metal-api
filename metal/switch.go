package metal

import (
	"time"
)

type Connections []Connection
type DeviceConnections map[string]Connections

// Switch have to register at the api. As metal-core is a stateless application running on a switch,
// the api needs persist all the information such that the core can create or restore a its entire
// switch configuration.
type Switch struct {
	ID          string            `json:"id" description:"a unique ID" unique:"true" modelDescription:"A switch that can register at the api." rethinkdb:"id,omitempty"`
	Nics        Nics              `json:"nics" description:"the list of network interfaces on the switch" rethinkdb:"network_interfaces"`
	Connections DeviceConnections `json:"connections" description:"a connection between a switch port and a device" rethinkdb:"connections"`
	SiteID      string            `json:"site_id" description:"the id of the site in which this switch is located" rethinkdb:"siteid"`
	RackID      string            `json:"rack_id" description:"the id of the rack in which this switch is located" rethinkdb:"rackid"`
	Created     time.Time         `json:"created" description:"the creation time of this switch" optional:"true" readOnly:"true" rethinkdb:"created"`
	Changed     time.Time         `json:"changed" description:"the last changed timestamp" optional:"true" readOnly:"true" rethinkdb:"changed"`
}

// Connection between switch port and device.
type Connection struct {
	Nic      Nic    `json:"nic" description:"a network interface on the switch" rethinkdb:"nic"`
	DeviceID string `json:"device_id,omitempty" optional:"true" description:"the device id of the device connected to the nic" rethinkdb:"deviceid"`
}

func NewSwitch(id, siteid, rackid string, nics Nics) *Switch {
	if nics == nil {
		nics = make([]Nic, 0)
	}
	return &Switch{
		ID:          id,
		SiteID:      siteid,
		RackID:      rackid,
		Connections: make(DeviceConnections),
		Nics:        nics,
	}
}

func (c Connections) ByNic() map[MacAddress]Connections {
	res := make(map[MacAddress]Connections)
	for _, con := range c {
		cons := res[con.Nic.MacAddress]
		cons = append(cons, con)
		res[con.Nic.MacAddress] = cons
	}
	return res
}

func (c Connections) ByDeviceID() map[string]Connections {
	res := make(map[string]Connections)
	for _, con := range c {
		cons := res[con.DeviceID]
		cons = append(cons, con)
		res[con.DeviceID] = cons
	}
	return res
}

// ConnectDevice iterates over all switch nics and device nic neighbor
// to find existing wire connections.
// Implementation is very inefficient, will also find all connections,
// which should not happen in a real environment.
func (s *Switch) ConnectDevice(device *Device) {
	// first remove all existing connections to this device.
	if _, has := s.Connections[device.ID]; has {
		delete(s.Connections, device.ID)
	}

	// calculate the connections for this device
	for _, switchNic := range s.Nics {
		for _, deviceNic := range device.Hardware.Nics {
			devNeighbors := deviceNic.Neighbors.ByMac()
			if _, has := devNeighbors[switchNic.MacAddress]; has {
				conn := Connection{
					Nic:      switchNic,
					DeviceID: device.ID,
				}
				s.Connections[device.ID] = append(s.Connections[device.ID], conn)
			}
		}
	}
}
