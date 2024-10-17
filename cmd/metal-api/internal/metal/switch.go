package metal

import (
	"fmt"
	"strconv"
	"strings"
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
	OS                 *SwitchOS     `rethinkdb:"os" json:"os"`
	ManagementIP       string        `rethinkdb:"management_ip" json:"management_ip"`
	ManagementUser     string        `rethinkdb:"management_user" json:"management_user"`
	ConsoleCommand     string        `rethinkdb:"console_command" json:"console_command"`
}

type Switches []Switch

type SwitchOS struct {
	Vendor           SwitchOSVendor `rethinkdb:"vendor" json:"vendor"`
	Version          string         `rethinkdb:"version" json:"version"`
	MetalCoreVersion string         `rethinkdb:"metal_core_version" json:"metal_core_version"`
}

// SwitchOSVendor is an enum denoting the name of a switch OS
type SwitchOSVendor string

// The enums for switch OS vendors
const (
	SwitchOSVendorSonic   SwitchOSVendor = "SONiC"
	SwitchOSVendorCumulus SwitchOSVendor = "Cumulus"
)

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

// SwitchStatus stores the received switch notifications in a separate table
type SwitchStatus struct {
	Base
	LastSync      *SwitchSync `rethinkdb:"last_sync" json:"last_sync" description:"last successful synchronization to the switch" optional:"true"`
	LastSyncError *SwitchSync `rethinkdb:"last_sync_error" json:"last_sync_error" description:"last synchronization to the switch that was erroneous" optional:"true"`
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

func ValidateSwitchOSVendor(os SwitchOSVendor) error {
	if os != SwitchOSVendorCumulus && os != SwitchOSVendorSonic {
		return fmt.Errorf("unknown switch os vendor %s", os)
	}
	return nil
}

// ByNicName builds a map of nic names to machine connection
func (c ConnectionMap) ByNicName() (map[string]Connection, error) {
	res := make(map[string]Connection)
	for _, cons := range c {
		for _, con := range cons {
			if _, has := res[con.Nic.Name]; has {
				return nil, fmt.Errorf("switch port %s is connected to more than one machine", con.Nic.Name)
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
			var has bool

			neighMap := machineNic.Neighbors.FilterByHostname(s.Name).ByIdentifier()

			_, has = neighMap[switchNic.GetIdentifier()]
			if has {
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

// SetVrfOfMachine set port on switch where machine is connected to given vrf
func (s *Switch) SetVrfOfMachine(m *Machine, vrf string) {
	affected := map[string]bool{}
	for _, c := range s.MachineConnections[m.ID] {
		affected[c.Nic.GetIdentifier()] = true
	}

	if len(affected) == 0 {
		return
	}

	nics := Nics{}
	for k, old := range s.Nics.ByIdentifier() {
		e := old
		if _, ok := affected[k]; ok {
			e.Vrf = vrf
		}
		nics = append(nics, *e)
	}
	s.Nics = nics
}

// TranslateNicMap creates a NicMap where the keys are translated to the naming convention of the target OS
//
// example mapping from cumulus to sonic for one single port:
//
//	map[string]Nic {
//		"swp1s1": Nic{
//			Name: "Ethernet1",
//			MacAddress: ""
//		}
//	}
func (s *Switch) TranslateNicMap(targetOS SwitchOSVendor) (NicMap, error) {
	nicMap := s.Nics.ByName()
	translatedNicMap := make(NicMap)

	if s.OS.Vendor == targetOS {
		return nicMap, nil
	}

	ports := make([]string, 0)
	for name := range nicMap {
		ports = append(ports, name)
	}

	lines, err := getLinesFromPortNames(ports, s.OS.Vendor)
	if err != nil {
		return nil, err
	}

	for _, p := range ports {
		targetPort, err := mapPortName(p, s.OS.Vendor, targetOS, lines)
		if err != nil {
			return nil, err
		}

		nic, ok := nicMap[p]
		if !ok {
			return nil, fmt.Errorf("an unknown error occured during port name translation")
		}
		translatedNicMap[targetPort] = nic
	}

	return translatedNicMap, nil
}

// MapPortNames creates a dictionary that maps the naming convention of this switch's OS to that of the target OS
func (s *Switch) MapPortNames(targetOS SwitchOSVendor) (map[string]string, error) {
	nics := s.Nics.ByName()
	portNamesMap := make(map[string]string, len(s.Nics))

	ports := make([]string, 0)
	for name := range nics {
		ports = append(ports, name)
	}

	lines, err := getLinesFromPortNames(ports, s.OS.Vendor)
	if err != nil {
		return nil, err
	}

	for _, p := range ports {
		targetPort, err := mapPortName(p, s.OS.Vendor, targetOS, lines)
		if err != nil {
			return nil, err
		}
		portNamesMap[p] = targetPort
	}

	return portNamesMap, nil
}

func mapPortName(port string, sourceOS, targetOS SwitchOSVendor, allLines []int) (string, error) {
	line, err := portNameToLine(port, sourceOS)
	if err != nil {
		return "", fmt.Errorf("unable to get line number from port name, %w", err)
	}

	if targetOS == SwitchOSVendorCumulus {
		return cumulusPortByLineNumber(line, allLines), nil
	}
	if targetOS == SwitchOSVendorSonic {
		return sonicPortByLineNumber(line), nil
	}

	return "", fmt.Errorf("unknown target switch os %s", targetOS)
}

func getLinesFromPortNames(ports []string, os SwitchOSVendor) ([]int, error) {
	lines := make([]int, 0)
	for _, p := range ports {
		l, err := portNameToLine(p, os)
		if err != nil {
			return nil, fmt.Errorf("unable to get line number from port name, %w", err)
		}

		lines = append(lines, l)
	}

	return lines, nil
}

func portNameToLine(port string, os SwitchOSVendor) (int, error) {
	if os == SwitchOSVendorSonic {
		return sonicPortNameToLine(port)
	}
	if os == SwitchOSVendorCumulus {
		return cumulusPortNameToLine(port)
	}
	return 0, fmt.Errorf("unknown switch os %s", os)
}

func sonicPortNameToLine(port string) (int, error) {
	// to prevent accidentally parsing a substring to a negative number
	if strings.Contains(port, "-") {
		return 0, fmt.Errorf("invalid token '-' in port name %s", port)
	}

	prefix, lineString, found := strings.Cut(port, "Ethernet")
	if !found {
		return 0, fmt.Errorf("invalid port name %s, expected to find prefix 'Ethernet'", port)
	}

	if prefix != "" {
		return 0, fmt.Errorf("invalid port name %s, port name is expected to start with 'Ethernet'", port)
	}

	line, err := strconv.Atoi(lineString)
	if err != nil {
		return 0, fmt.Errorf("unable to convert port name to line number: %w", err)
	}

	return line, nil
}

func cumulusPortNameToLine(port string) (int, error) {
	// to prevent accidentally parsing a substring to a negative number
	if strings.Contains(port, "-") {
		return 0, fmt.Errorf("invalid token '-' in port name %s", port)
	}

	prefix, suffix, found := strings.Cut(port, "swp")
	if !found {
		return 0, fmt.Errorf("invalid port name %s, expected to find prefix 'swp'", port)
	}

	if prefix != "" {
		return 0, fmt.Errorf("invalid port name %s, port name is expected to start with 'swp'", port)
	}

	var line int

	countString, indexString, found := strings.Cut(suffix, "s")
	if !found {
		count, err := strconv.Atoi(suffix)
		if err != nil {
			return 0, fmt.Errorf("unable to convert port name to line number: %w", err)
		}
		if count <= 0 {
			return 0, fmt.Errorf("invalid port name %s would map to negative number", port)
		}
		line = (count - 1) * 4
	} else {
		count, err := strconv.Atoi(countString)
		if err != nil {
			return 0, fmt.Errorf("unable to convert port name to line number: %w", err)
		}
		if count <= 0 {
			return 0, fmt.Errorf("invalid port name %s would map to negative number", port)
		}

		index, err := strconv.Atoi(indexString)
		if err != nil {
			return 0, fmt.Errorf("unable to convert port name to line number: %w", err)
		}
		line = (count-1)*4 + index
	}

	return line, nil
}

func sonicPortByLineNumber(line int) string {
	return fmt.Sprintf("Ethernet%d", line)
}

func cumulusPortByLineNumber(line int, allLines []int) string {
	if line%4 > 0 {
		return fmt.Sprintf("swp%ds%d", line/4+1, line%4)
	}

	for _, l := range allLines {
		if l == line {
			continue
		}
		if l/4 == line/4 {
			return fmt.Sprintf("swp%ds%d", line/4+1, line%4)
		}
	}

	return fmt.Sprintf("swp%d", line/4+1)
}
