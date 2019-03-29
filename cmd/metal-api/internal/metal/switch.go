package metal

// Switch have to register at the api. As metal-core is a stateless application running on a switch,
// the api needs persist all the information such that the core can create or restore a its entire
// switch configuration.
type Switch struct {
	Base
	Nics               Nics          `json:"nics" modelDescription:"A switch that can register at the api." description:"the list of network interfaces on the switch" rethinkdb:"network_interfaces"`
	Connections        Connections   `json:"connections" description:"a connection between a switch port and a machine" rethinkdb:"-"`
	MachineConnections ConnectionMap `json:"-" description:"a connection between a switch port and a machine" rethinkdb:"machineconnections"`
	PartitionID        string        `json:"partition_id" description:"the id of the partition in which this switch is located" rethinkdb:"partitionid"`
	RackID             string        `json:"rack_id" description:"the id of the rack in which this switch is located" rethinkdb:"rackid"`
	Partition          Partition     `json:"partition" description:"the partition in which this switch is located" rethinkdb:"-"`
}

// Connection between switch port and machine.
type Connection struct {
	Nic       Nic    `json:"nic" description:"a network interface on the switch" rethinkdb:"nic"`
	MachineID string `json:"machine_id,omitempty" optional:"true" description:"the machine id of the machine connected to the nic" rethinkdb:"machineid"`
}

// Connections is a list of connections.
type Connections []Connection

// ConnectionMap is an indexed map of connection-lists
type ConnectionMap map[string]Connections

// SwitchEvent is propagated when a switch needs to update its configuration.
type SwitchEvent struct {
	Type     EventType `json:"type"`
	Machine  Machine   `json:"machine"`
	Switches []Switch  `json:"switches"`
}

// NewSwitch creates a new switch with the given data fields.
func NewSwitch(id, rackid string, nics Nics, part *Partition) *Switch {
	if nics == nil {
		nics = make([]Nic, 0)
	}
	return &Switch{
		Base: Base{
			ID:      id,
			Name:    id,
			Created: getNow(),
			Changed: getNow(),
		},
		PartitionID:        part.ID,
		RackID:             rackid,
		Connections:        make([]Connection, 0),
		MachineConnections: make(ConnectionMap),
		Nics:               nics,
		Partition:          *part,
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

// ConnectMachine iterates over all switch nics and machine nic neighbor
// to find existing wire connections.
// Implementation is very inefficient, will also find all connections,
// which should not happen in a real environment.
func (s *Switch) ConnectMachine(machine *Machine) {
	// first remove all existing connections to this machine.
	if _, has := s.MachineConnections[machine.ID]; has {
		delete(s.MachineConnections, machine.ID)
	}

	// calculate the connections for this machine
	for _, switchNic := range s.Nics {
		for _, machineNic := range machine.Hardware.Nics {
			devNeighbors := machineNic.Neighbors.ByMac()
			if _, has := devNeighbors[switchNic.MacAddress]; has {
				conn := Connection{
					Nic:       switchNic,
					MachineID: machine.ID,
				}
				s.MachineConnections[machine.ID] = append(s.MachineConnections[machine.ID], conn)
			}
		}
	}
}

// FillSwitchConnections fills the Connections field of the switch by
// iterating over the connections in the MachineConnections which are
// stored in the database.
func (s *Switch) FillSwitchConnections() {
	cons := make(Connections, 0)
	for _, cc := range s.MachineConnections {
		for _, c := range cc {
			cons = append(cons, c)
		}
	}
	s.Connections = cons
}

// FromSwitch stores the machine connections from another switch in the new instance.
func (s *Switch) FromSwitch(other *Switch) {
	s.MachineConnections = other.MachineConnections
	s.FillSwitchConnections()
}

// FillAllConnections fills all Connections of all given switches.
func FillAllConnections(sw []Switch) {
	for i := range sw {
		s := &sw[i]
		s.FillSwitchConnections()
	}
}
