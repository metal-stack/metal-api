package metal

import (
	"fmt"
	"time"
)

// Switch have to register at the api. As metal-core is a stateless application running on a switch,
// the api needs persist all the information such that the core can create or restore a its entire
// switch configuration.
type Switch struct {
	Base
	Nics               Nics          `rethinkdb:"network_interfaces" json:"network_interfaces"`
	MachineConnections ConnectionMap `rethinkdb:"machineconnections" json:"machineconnections"`
	PartitionID        string        `rethinkdb:"partitionid" json:"partitionid"`
	RackID             string        `rethinkdb:"rackid" json:"rackid"`
	Mode               SwitchMode    `rethinkdb:"mode" json:"mode"`
	LastSync           *SwitchSync   `rethinkdb:"last_sync" json:"last_sync"`
	LastSyncError      *SwitchSync   `rethinkdb:"last_sync_error" json:"last_sync_error"`
}

// Connection between switch port and machine.
type Connection struct {
	Nic       Nic    `rethinkdb:"nic" json:"nic"`
	MachineID string `rethinkdb:"machineid" json:"machineid"`
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

// SwitchSync contains information about the last synchronization of the state held in the metal-api to a switch.
type SwitchSync struct {
	Time     time.Time     `rethinkdb:"time" json:"time"`
	Duration time.Duration `rethinkdb:"duration" json:"duration"`
	Error    *string       `rethinkdb:"error" json:"error"`
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

// ByNicName builds a map of nic names to machine connection
func (c ConnectionMap) ByNicName() (map[string]Connection, error) {
	res := make(map[string]Connection)
	for _, cons := range c {
		for _, con := range cons {
			if con2, has := res[con.Nic.Name]; has {
				return nil, fmt.Errorf("connection map has duplicate connections for nic %s; con1: %v, con2: %v", con.Nic.Name, con, con2)
			}
			res[con.Nic.Name] = con
		}
	}
	return res, nil
}

// ConnectMachine iterates over all switch nics and machine nic neighbor
// to find existing wire connections.
// Implementation is very inefficient, will also find all connections,
// which should not happen in a real environment.
func (s *Switch) ConnectMachine(machine *Machine) int {
	// first remove all existing connections to this machine.
	delete(s.MachineConnections, machine.ID)

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
	return len(s.MachineConnections[machine.ID])
}
