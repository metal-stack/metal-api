package metal

// Switch have to register at the api. As metal-core is a stateless application running on a switch,
// the api needs persist all the information such that the core can create or restore a its entire
// switch configuration.
type Switch struct {
	Base
	Nics              Nics          `json:"nics" modelDescription:"A switch that can register at the api." description:"the list of network interfaces on the switch" rethinkdb:"network_interfaces"`
	Connections       Connections   `json:"connections" description:"a connection between a switch port and a device" rethinkdb:"-"`
	DeviceConnections ConnectionMap `json:"-" description:"a connection between a switch port and a device" rethinkdb:"deviceconnections"`
	PartitionID       string        `json:"partition_id" description:"the id of the partition in which this switch is located" rethinkdb:"partitionid"`
	RackID            string        `json:"rack_id" description:"the id of the rack in which this switch is located" rethinkdb:"rackid"`
	Partition         Partition     `json:"partition" description:"the partition in which this switch is located" rethinkdb:"-"`
}

// Connection between switch port and device.
type Connection struct {
	Nic      Nic    `json:"nic" description:"a network interface on the switch" rethinkdb:"nic"`
	DeviceID string `json:"device_id,omitempty" optional:"true" description:"the device id of the device connected to the nic" rethinkdb:"deviceid"`
}

// Connections is a list of connections.
type Connections []Connection

// ConnectionMap is an indexed map of connection-lists
type ConnectionMap map[string]Connections

// NewSwitch creates a new switch with the given data fields.
func NewSwitch(id, rackid string, nics Nics, part *Partition) *Switch {
	if nics == nil {
		nics = make([]Nic, 0)
	}
	return &Switch{
		Base: Base{
			ID:   id,
			Name: id,
		},
		PartitionID:       part.ID,
		RackID:            rackid,
		Connections:       make([]Connection, 0),
		DeviceConnections: make(ConnectionMap),
		Nics:              nics,
		Partition:         *part,
	}
}

// ByNic creates a map of connections-lists with the MAC adress as the index.
func (c Connections) ByNic() map[MacAddress]Connections {
	res := make(map[MacAddress]Connections)
	for _, con := range c {
		cons := res[con.Nic.MacAddress]
		cons = append(cons, con)
		res[con.Nic.MacAddress] = cons
	}
	return res
}

// ConnectDevice iterates over all switch nics and device nic neighbor
// to find existing wire connections.
// Implementation is very inefficient, will also find all connections,
// which should not happen in a real environment.
func (s *Switch) ConnectDevice(device *Device) {
	// first remove all existing connections to this device.
	if _, has := s.DeviceConnections[device.ID]; has {
		delete(s.DeviceConnections, device.ID)
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
				s.DeviceConnections[device.ID] = append(s.DeviceConnections[device.ID], conn)
			}
		}
	}
}

// FillSwitchConnections fills the Connections field of the switch by
// iterating over the connections in the DeviceConnections which are
// stored in the database.
func (s *Switch) FillSwitchConnections() {
	cons := make(Connections, 0)
	for _, cc := range s.DeviceConnections {
		for _, c := range cc {
			cons = append(cons, c)
		}
	}
	s.Connections = cons
}

// FillAllConnections fills all Connections of all given switches.
func FillAllConnections(sw []Switch) {
	for i := range sw {
		s := &sw[i]
		s.FillSwitchConnections()
	}
}
