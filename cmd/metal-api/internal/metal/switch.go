package metal

// Switch have to register at the api. As metal-core is a stateless application running on a switch,
// the api needs persist all the information such that the core can create or restore a its entire
// switch configuration.
type Switch struct {
	Base
	Nics               Nics          `rethinkdb:"network_interfaces"`
	MachineConnections ConnectionMap `rethinkdb:"machineconnections"`
	PartitionID        string        `rethinkdb:"partitionid"`
	RackID             string        `rethinkdb:"rackid"`
	Mode               SwitchMode    `rethinkdb:"mode"`
}

// Connection between switch port and machine.
type Connection struct {
	Nic       Nic    `rethinkdb:"nic"`
	MachineID string `rethinkdb:"machineid"`
}

// Connections is a list of connections.
type Connections []Connection

// ConnectionMap is an indexed map of connection-lists
type ConnectionMap map[string]Connections

// A SwitchMode is an enum which indicates the mode of a switch
type SwitchMode string

// The enums for the switch modes.
const (
	SwitchOperational SwitchMode = "operational"
	SwitchReplace     SwitchMode = "replace"
)

// SwitchEvent is propagated when a switch needs to update its configuration.
type SwitchEvent struct {
	Type     EventType `json:"type"`
	Machine  Machine   `json:"machine"`
	Switches []Switch  `json:"switches"`
}

// SwitchModeFrom converts a switch mode string to the type
func SwitchModeFrom(name string) SwitchMode {
	switch name {
	case string(SwitchReplace):
		return SwitchReplace
	default:
		return SwitchOperational
	}
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
