package metal

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/dustin/go-humanize"
	mn "github.com/metal-stack/metal-lib/pkg/net"
	"github.com/samber/lo"
)

// A MState is an enum which indicates the state of a machine
type MState string

// Role describes the role of a machine.
type Role string

const (
	// AvailableState describes a machine state where a machine is available for an allocation
	AvailableState MState = ""
	// ReservedState describes a machine state where a machine is not being considered for random allocation
	ReservedState MState = "RESERVED"
	// LockedState describes a machine state where a machine cannot be deleted or allocated anymore
	LockedState MState = "LOCKED"
)

var (
	// RoleMachine is a role that indicates the allocated machine acts as a machine
	RoleMachine Role = "machine"
	// RoleFirewall is a role that indicates the allocated machine acts as a firewall
	RoleFirewall Role = "firewall"
)

var (
	// AllStates contains all possible values of a machine state
	AllStates = []MState{AvailableState, ReservedState, LockedState}
	// AllRoles contains all possible values of a role
	AllRoles = map[Role]bool{
		RoleMachine:  true,
		RoleFirewall: true,
	}
)

// A MachineState describes the state of a machine. If the Value is AvailableState,
// the machine will be available for allocation. In all other cases the allocation
// must explicitly point to this machine.
type MachineState struct {
	Value              MState `rethinkdb:"value" json:"value"`
	Description        string `rethinkdb:"description" json:"description"`
	Issuer             string `rethinkdb:"issuer" json:"issuer,omitempty"`
	MetalHammerVersion string `rethinkdb:"metal_hammer_version" json:"metal_hammer_version"`
}

// MachineStateFrom converts a machineState string to the type
func MachineStateFrom(name string) (MState, error) {
	switch name {
	case string(AvailableState):
		return AvailableState, nil
	case string(ReservedState):
		return ReservedState, nil
	case string(LockedState):
		return LockedState, nil
	default:
		return "", fmt.Errorf("unknown MachineState:%s", name)
	}
}

// LEDState is the state of the LED of the Machine
type LEDState string

const (
	// LEDStateOn LED is on
	LEDStateOn LEDState = "LED-ON"
	// LEDStateOff LED is off
	LEDStateOff LEDState = "LED-OFF"
)

// LEDStateFrom converts an LEDState string to the corresponding type
func LEDStateFrom(name string) (LEDState, error) {
	switch name {
	case string(LEDStateOff):
		return LEDStateOff, nil
	case string(LEDStateOn):
		return LEDStateOn, nil
	default:
		return "", fmt.Errorf("unknown LEDState:%s", name)
	}
}

// A ChassisIdentifyLEDState describes the state of a chassis identify LED, i.e. LED-ON/LED-OFF.
type ChassisIdentifyLEDState struct {
	Value       LEDState `rethinkdb:"value" json:"value"`
	Description string   `rethinkdb:"description" json:"description"`
}

// A Machine is a piece of metal which is under the control of our system. It registers itself
// and can be allocated or freed. If the machine is allocated, the substructure Allocation will
// be filled. Any unallocated (free) machine won't have such values.
type Machine struct {
	Base
	Allocation   *MachineAllocation      `rethinkdb:"allocation" json:"allocation"`
	PartitionID  string                  `rethinkdb:"partitionid" json:"partitionid"`
	SizeID       string                  `rethinkdb:"sizeid" json:"sizeid"`
	RackID       string                  `rethinkdb:"rackid" json:"rackid"`
	Waiting      bool                    `rethinkdb:"waiting" json:"waiting"`
	PreAllocated bool                    `rethinkdb:"preallocated" json:"preallocated"`
	Hardware     MachineHardware         `rethinkdb:"hardware" json:"hardware"`
	State        MachineState            `rethinkdb:"state" json:"state"`
	LEDState     ChassisIdentifyLEDState `rethinkdb:"ledstate" json:"ledstate"`
	Tags         []string                `rethinkdb:"tags" json:"tags"`
	IPMI         IPMI                    `rethinkdb:"ipmi" json:"ipmi"`
	BIOS         BIOS                    `rethinkdb:"bios" json:"bios"`
}

// Machines is a slice of Machine
type Machines []Machine

// IsFirewall returns true if this machine is a firewall machine.
func (m *Machine) IsFirewall() bool {
	if m.Allocation == nil {
		return false
	}
	if m.Allocation.Role == RoleFirewall {
		return true
	}
	return false
}

// A MachineAllocation stores the data which are only present for allocated machines.
type MachineAllocation struct {
	Creator          string            `rethinkdb:"creator" json:"creator"`
	Created          time.Time         `rethinkdb:"created" json:"created"`
	Name             string            `rethinkdb:"name" json:"name"`
	Description      string            `rethinkdb:"description" json:"description"`
	Project          string            `rethinkdb:"project" json:"project"`
	ImageID          string            `rethinkdb:"imageid" json:"imageid"`
	FilesystemLayout *FilesystemLayout `rethinkdb:"filesystemlayout" json:"filesystemlayout"`
	MachineNetworks  []*MachineNetwork `rethinkdb:"networks" json:"networks"`
	Hostname         string            `rethinkdb:"hostname" json:"hostname"`
	SSHPubKeys       []string          `rethinkdb:"sshPubKeys" json:"sshPubKeys"`
	UserData         string            `rethinkdb:"userdata" json:"userdata"`
	ConsolePassword  string            `rethinkdb:"console_password" json:"console_password"`
	Succeeded        bool              `rethinkdb:"succeeded" json:"succeeded"`
	Reinstall        bool              `rethinkdb:"reinstall" json:"reinstall"`
	MachineSetup     *MachineSetup     `rethinkdb:"setup" json:"setup"`
	Role             Role              `rethinkdb:"role" json:"role"`
	VPN              *MachineVPN       `rethinkdb:"vpn" json:"vpn"`
	UUID             string            `rethinkdb:"uuid" json:"uuid"`
	FirewallRules    *FirewallRules    `rethinkdb:"firewall_rules" json:"firewall_rules"`
	DNSServers       DNSServers        `rethinkdb:"dns_servers" json:"dns_servers"`
	NTPServers       NTPServers        `rethinkdb:"ntp_servers" json:"ntp_servers"`
}

type FirewallRules struct {
	Egress  []EgressRule  `rethinkdb:"egress" json:"egress"`
	Ingress []IngressRule `rethinkdb:"ingress" json:"ingress"`
}

type EgressRule struct {
	Protocol Protocol `rethinkdb:"protocol" json:"protocol"`
	Ports    []int    `rethinkdb:"ports" json:"ports"`
	To       []string `rethinkdb:"to" json:"to"`
	Comment  string   `rethinkdb:"comment" json:"comment"`
}

type IngressRule struct {
	Protocol Protocol `rethinkdb:"protocol" json:"protocol"`
	Ports    []int    `rethinkdb:"ports" json:"ports"`
	To       []string `rethinkdb:"to" json:"to"`
	From     []string `rethinkdb:"from" json:"from"`
	Comment  string   `rethinkdb:"comment" json:"comment"`
}

type DNSServers []DNSServer

type DNSServer struct {
	IP string `rethinkdb:"ip" json:"ip" description:"ip address of this dns server"`
}

type NTPServers []NTPServer

type NTPServer struct {
	Address string `address:"address" json:"address" description:"ip address or dns hostname of this ntp server"`
}

type Protocol string

const (
	ProtocolTCP Protocol = "TCP"
	ProtocolUDP Protocol = "UDP"
)

func ProtocolFromString(s string) (Protocol, error) {
	switch strings.ToLower(s) {
	case "tcp":
		return ProtocolTCP, nil
	case "udp":
		return ProtocolUDP, nil
	default:
		return Protocol(""), fmt.Errorf("no such protocol: %s", s)
	}
}

func (r EgressRule) Validate() error {
	switch r.Protocol {
	case ProtocolTCP, ProtocolUDP:
		// ok
	default:
		return fmt.Errorf("invalid protocol: %s", r.Protocol)
	}

	if err := validateComment(r.Comment); err != nil {
		return err
	}
	if err := validatePorts(r.Ports); err != nil {
		return err
	}

	if err := validateCIDRs(r.To); err != nil {
		return err
	}

	return nil
}

func (r IngressRule) Validate() error {
	switch r.Protocol {
	case ProtocolTCP, ProtocolUDP:
		// ok
	default:
		return fmt.Errorf("invalid protocol: %s", r.Protocol)
	}
	if err := validateComment(r.Comment); err != nil {
		return err
	}

	if err := validatePorts(r.Ports); err != nil {
		return err
	}
	if err := validateCIDRs(r.To); err != nil {
		return err
	}
	if err := validateCIDRs(r.From); err != nil {
		return err
	}
	if err := validateCIDRs(slices.Concat(r.From, r.To)); err != nil {
		return err
	}

	return nil
}

const (
	allowedCharacters = "abcdefghijklmnopqrstuvwxyz_- "
	maxCommentLength  = 100
)

func validateComment(comment string) error {
	for _, c := range comment {
		if !strings.Contains(allowedCharacters, strings.ToLower(string(c))) {
			return fmt.Errorf("illegal character in comment found, only: %q allowed", allowedCharacters)
		}
	}
	if len(comment) > maxCommentLength {
		return fmt.Errorf("comments can not exceed %d characters", maxCommentLength)
	}
	return nil
}

func validatePorts(ports []int) error {
	for _, port := range ports {
		if port < 0 || port > 65535 {
			return fmt.Errorf("port is out of range")
		}
	}

	return nil
}

func validateCIDRs(cidrs []string) error {
	af := ""
	for _, cidr := range cidrs {
		p, err := netip.ParsePrefix(cidr)
		if err != nil {
			return fmt.Errorf("invalid cidr: %w", err)
		}
		var newaf string
		if p.Addr().Is4() {
			newaf = "ipv4"
		} else {
			newaf = "ipv6"
		}
		if af != "" && af != newaf {
			return fmt.Errorf("mixed address family in one rule is not supported:%v", cidrs)
		}
		af = newaf
	}
	return nil
}

// A MachineSetup stores the data used for machine reinstallations.
type MachineSetup struct {
	ImageID      string `rethinkdb:"imageid" json:"imageid"`
	PrimaryDisk  string `rethinkdb:"primarydisk" json:"primarydisk"`
	OSPartition  string `rethinkdb:"ospartition" json:"ospartition"`
	Initrd       string `rethinkdb:"initrd" json:"initrd"`
	Cmdline      string `rethinkdb:"cmdline" json:"cmdline"`
	Kernel       string `rethinkdb:"kernel" json:"kernel"`
	BootloaderID string `rethinkdb:"bootloaderid" json:"bootloaderid"`
}

// ByProjectID creates a map of machines with the project id as the index.
func (ms Machines) ByProjectID() map[string]Machines {
	res := make(map[string]Machines)
	for i, m := range ms {
		if m.Allocation != nil {
			res[m.Allocation.Project] = append(res[m.Allocation.Project], ms[i])
		}
	}
	return res
}

func (ms Machines) WithSize(id string) Machines {
	var res Machines

	for _, m := range ms {
		m := m

		if m.SizeID != id {
			continue
		}

		res = append(res, m)
	}

	return res
}

func (ms Machines) WithPartition(id string) Machines {
	var res Machines

	for _, m := range ms {
		m := m

		if m.PartitionID != id {
			continue
		}

		res = append(res, m)
	}

	return res
}

// MachineNetwork stores the Network details of the machine
type MachineNetwork struct {
	NetworkID           string   `rethinkdb:"networkid" json:"networkid"`
	Prefixes            []string `rethinkdb:"prefixes" json:"prefixes"`
	IPs                 []string `rethinkdb:"ips" json:"ips"`
	DestinationPrefixes []string `rethinkdb:"destinationprefixes" json:"destinationprefixes"`
	Vrf                 uint     `rethinkdb:"vrf" json:"vrf"`
	PrivatePrimary      bool     `rethinkdb:"privateprimary" json:"privateprimary"`
	Private             bool     `rethinkdb:"private" json:"private"`
	ASN                 uint32   `rethinkdb:"asn" json:"asn"`
	Nat                 bool     `rethinkdb:"nat" json:"nat"`
	Underlay            bool     `rethinkdb:"underlay" json:"underlay"`
	Shared              bool     `rethinkdb:"shared" json:"shared"`
}

// NetworkType represents the type of a network
type NetworkType struct {
	Name           string `json:"name,omitempty"`
	Private        bool   `json:"private,omitempty"`
	PrivatePrimary bool   `json:"private_primary,omitempty"`
	Shared         bool   `json:"shared,omitempty"`
	Underlay       bool   `json:"underlay,omitempty"`
	Supported      bool   `json:"-"`
}

var (
	PrivatePrimaryUnshared = NetworkType{
		Name:           mn.PrivatePrimaryUnshared,
		Private:        true,
		PrivatePrimary: true,
		Shared:         false,
		Underlay:       false,
		Supported:      true,
	}
	PrivatePrimaryShared = NetworkType{
		Name:           mn.PrivatePrimaryShared,
		Private:        true,
		PrivatePrimary: true,
		Shared:         true,
		Underlay:       false,
		Supported:      true,
	}
	PrivateSecondaryShared = NetworkType{
		Name:           mn.PrivateSecondaryShared,
		Private:        true,
		PrivatePrimary: false,
		Shared:         true,
		Underlay:       false,
		Supported:      true,
	}
	// PrivateSecondaryUnshared this case is not a valid configuration
	PrivateSecondaryUnshared = NetworkType{
		Name:           mn.PrivateSecondaryUnshared,
		Private:        true,
		PrivatePrimary: false,
		Shared:         false,
		Underlay:       false,
		Supported:      false,
	}
	External = NetworkType{
		Name:           mn.External,
		Private:        false,
		PrivatePrimary: false,
		Shared:         false,
		Underlay:       false,
		Supported:      true,
	}
	Underlay = NetworkType{
		Name:           mn.Underlay,
		Private:        false,
		PrivatePrimary: false,
		Shared:         false,
		Underlay:       true,
		Supported:      true,
	}
	AllNetworkTypes = []NetworkType{PrivatePrimaryUnshared, PrivatePrimaryShared, PrivateSecondaryShared, PrivateSecondaryUnshared, External, Underlay}
)

// Is checks whether the machine network has the given type
func (mn *MachineNetwork) Is(n NetworkType) bool {
	return mn.Private == n.Private && mn.PrivatePrimary == n.PrivatePrimary && mn.Shared == n.Shared && mn.Underlay == n.Underlay
}

// NetworkType determines the network type based on the flags stored in the db entity.
func (mn *MachineNetwork) NetworkType() (*NetworkType, error) {
	var nt *NetworkType
	for i := range AllNetworkTypes {
		t := AllNetworkTypes[i]
		if mn.Is(t) {
			nt = &t
			break
		}
	}
	if nt == nil {
		return nil, fmt.Errorf("could not determine network type out of flags, underlay: %v, privateprimary: %v, private: %v, shared: %v", mn.Underlay, mn.PrivatePrimary, mn.Private, mn.Shared)
	}

	if nt.Supported {
		return nt, nil
	}
	// This is for machineNetworks from an allocation which was before NetworkType was introduced.
	// We guess based on unset fields not present at this time and therefore are set to false.
	// TODO: This guess based approach can be removed in future releases.
	if mn.Private && !mn.PrivatePrimary && !mn.Shared && !mn.Underlay {
		return &PrivatePrimaryUnshared, nil
	}
	return nil, fmt.Errorf("determined network type out of flags, underlay: %v, privateprimary: %v, private: %v, shared: %v is unsupported", mn.Underlay, mn.PrivatePrimary, mn.Private, mn.Shared)
}

func (n NetworkType) String() string {
	return n.Name
}

// MachineHardware stores the data which is collected by our system on the hardware when it registers itself.
type MachineHardware struct {
	Memory    uint64        `rethinkdb:"memory" json:"memory"`
	Nics      Nics          `rethinkdb:"network_interfaces" json:"network_interfaces"`
	Disks     []BlockDevice `rethinkdb:"block_devices" json:"block_devices"`
	MetalCPUs []MetalCPU    `rethinkdb:"cpus" json:"cpus"`
	MetalGPUs []MetalGPU    `rethinkdb:"gpus" json:"gpus"`
}

type MetalCPU struct {
	Vendor  string `rethinkdb:"vendor" json:"vendor"`
	Model   string `rethinkdb:"model" json:"model"`
	Cores   uint32 `rethinkdb:"cores" json:"cores"`
	Threads uint32 `rethinkdb:"threads" json:"threads"`
}

type MetalGPU struct {
	Vendor string `rethinkdb:"vendor" json:"vendor"`
	Model  string `rethinkdb:"model" json:"model"`
}

// MachineLiveliness indicates the liveliness of a machine
type MachineLiveliness string

// The enums for the machine liveliness states.
const (
	MachineLivelinessAlive   MachineLiveliness = "Alive"
	MachineLivelinessDead    MachineLiveliness = "Dead"
	MachineLivelinessUnknown MachineLiveliness = "Unknown"
	MachineDeadAfter         time.Duration     = 5 * time.Minute
	MachineResurrectAfter    time.Duration     = time.Hour
)

func capacityOf[V any](identifier string, vs []V, countFn func(v V) (model string, count uint64)) (uint64, []V) {
	var (
		sum     uint64
		matched []V
	)

	for _, v := range vs {
		model, count := countFn(v)

		if identifier != "" {
			matches, err := filepath.Match(identifier, model)
			if err != nil {
				// illegal identifiers are already prevented by size validation
				continue
			}

			if !matches {
				continue
			}
		}

		sum += count
		matched = append(matched, v)
	}

	return sum, matched
}

func exhaustiveMatch[V comparable](cs []Constraint, vs []V, countFn func(v V) (model string, count uint64)) bool {
	unmatched := slices.Clone(vs)

	for _, c := range cs {
		capacity, matched := capacityOf(c.Identifier, vs, countFn)

		match := c.inRange(capacity)
		if !match {
			continue
		}

		unmatched, _ = lo.Difference(unmatched, matched)
	}

	return len(unmatched) == 0
}

// ReadableSpec returns a human readable string for the hardware.
func (hw *MachineHardware) ReadableSpec() string {
	diskCapacity, _ := capacityOf("*", hw.Disks, countDisk)
	cpus, _ := capacityOf("*", hw.MetalCPUs, countCPU)
	gpus, _ := capacityOf("*", hw.MetalGPUs, countGPU)
	return fmt.Sprintf("CPUs: %d, Memory: %s, Storage: %s, GPUs: %d", cpus, humanize.Bytes(hw.Memory), humanize.Bytes(diskCapacity), gpus)
}

// BlockDevice information.
type BlockDevice struct {
	Name string `rethinkdb:"name" json:"name"`
	Size uint64 `rethinkdb:"size" json:"size"`
}

// Fru (Field Replaceable Unit) data
type Fru struct {
	ChassisPartNumber   string `rethinkdb:"chassis_part_number" json:"chassis_part_number"`
	ChassisPartSerial   string `rethinkdb:"chassis_part_serial" json:"chassis_part_serial"`
	BoardMfg            string `rethinkdb:"board_mfg" json:"board_mfg"`
	BoardMfgSerial      string `rethinkdb:"board_mfg_serial" json:"board_mfg_serial"`
	BoardPartNumber     string `rethinkdb:"board_part_number" json:"board_part_number"`
	ProductManufacturer string `rethinkdb:"product_manufacturer" json:"product_manufacturer"`
	ProductPartNumber   string `rethinkdb:"product_part_number" json:"product_part_number"`
	ProductSerial       string `rethinkdb:"product_serial" json:"product_serial"`
}

// IPMI connection data
type IPMI struct {
	// Address is host:port of the connection to the ipmi BMC, host can be either a ip address or a hostname
	Address       string        `rethinkdb:"address" json:"address"`
	MacAddress    string        `rethinkdb:"mac" json:"mac"`
	User          string        `rethinkdb:"user" json:"user"`
	Password      string        `rethinkdb:"password" json:"password"`
	Interface     string        `rethinkdb:"interface" json:"interface"`
	Fru           Fru           `rethinkdb:"fru" json:"fru"`
	BMCVersion    string        `rethinkdb:"bmcversion" json:"bmcversion"`
	PowerState    string        `rethinkdb:"powerstate" json:"powerstate"`
	PowerMetric   *PowerMetric  `rethinkdb:"powermetric" json:"powermetric"`
	PowerSupplies PowerSupplies `rethinkdb:"powersupplies" json:"powersupplies"`
	LastUpdated   time.Time     `rethinkdb:"last_updated" json:"last_updated"`
}

type PowerMetric struct {
	// AverageConsumedWatts shall represent the
	// average power level that occurred averaged over the last IntervalInMin
	// minutes.
	AverageConsumedWatts float32 `rethinkdb:"averageconsumedwatts" json:"averageconsumedwatts"`
	// IntervalInMin shall represent the time
	// interval (or window), in minutes, in which the PowerMetrics properties
	// are measured over.
	// Should be an integer, but some Dell implementations return as a float.
	IntervalInMin float32 `rethinkdb:"intervalinmin" json:"intervalinmin"`
	// MaxConsumedWatts shall represent the
	// maximum power level in watts that occurred within the last
	// IntervalInMin minutes.
	MaxConsumedWatts float32 `rethinkdb:"maxconsumedwatts" json:"maxconsumedwatts"`
	// MinConsumedWatts shall represent the
	// minimum power level in watts that occurred within the last
	// IntervalInMin minutes.
	MinConsumedWatts float32 `rethinkdb:"minconsumedwatts" json:"minconsumedwatts"`
}

type PowerSupplies []PowerSupply
type PowerSupply struct {
	// Status shall contain any status or health properties
	// of the resource.
	Status PowerSupplyStatus `rethinkdb:"status" json:"status"`
}
type PowerSupplyStatus struct {
	Health string `rethinkdb:"health" json:"health"`
	State  string `rethinkdb:"state" json:"state"`
}

// BIOS contains machine bios information
type BIOS struct {
	Version string `rethinkdb:"version" json:"version"`
	Vendor  string `rethinkdb:"vendor" json:"vendor"`
	Date    string `rethinkdb:"date" json:"date"`
}

// HasMAC returns true if this machine has the given MAC.
func (m *Machine) HasMAC(mac string) bool {
	for _, nic := range m.Hardware.Nics {
		if string(nic.MacAddress) == mac {
			return true
		}
	}
	return false
}

// A MachineCommand is an alias of a string
type MachineCommand string

// our supported machines commands.
const (
	MachineOnCmd             MachineCommand = "ON"
	MachineOffCmd            MachineCommand = "OFF"
	MachineResetCmd          MachineCommand = "RESET"
	MachineCycleCmd          MachineCommand = "CYCLE"
	MachineBiosCmd           MachineCommand = "BIOS"
	MachineDiskCmd           MachineCommand = "DISK"
	MachinePxeCmd            MachineCommand = "PXE"
	MachineReinstallCmd      MachineCommand = "REINSTALL"
	ChassisIdentifyLEDOnCmd  MachineCommand = "LED-ON"
	ChassisIdentifyLEDOffCmd MachineCommand = "LED-OFF"
	UpdateFirmwareCmd        MachineCommand = "UPDATE-FIRMWARE"
)

// A MachineExecCommand can be sent via a MachineEvent to execute
// the command against the specific machine. The specified command
// should be executed against the given target machine. The parameters
// is an optional array of strings which are implementation specific
// and dependent of the command.
type MachineExecCommand struct {
	TargetMachineID string          `json:"target,omitempty"`
	Command         MachineCommand  `json:"cmd,omitempty"`
	IPMI            *IPMI           `json:"ipmi,omitempty"`
	FirmwareUpdate  *FirmwareUpdate `json:"firmwareupdate,omitempty"`
}

// MachineEvent is propagated when a machine is create/updated/deleted.
type MachineEvent struct {
	Type         EventType           `json:"type,omitempty"`
	OldMachineID string              `json:"old,omitempty"`
	NewMachineID string              `json:"new,omitempty"`
	Cmd          *MachineExecCommand `json:"cmd,omitempty"`
}

// AllocationEvent is propagated when a machine is allocated.
type AllocationEvent struct {
	MachineID string `json:"old,omitempty"`
}

type FirmwareUpdate struct {
	Kind FirmwareKind `json:"kind"`
	URL  string       `json:"url"`
}

type MachineVPN struct {
	ControlPlaneAddress string `rethinkdb:"address" json:"address"`
	AuthKey             string `rethinkdb:"auth_key" json:"auth_key"`
	Connected           bool   `rethinkdb:"connected" json:"connected"`
}

type MachineIPMISuperUser struct {
	password string
}

func NewIPMISuperUser(log *slog.Logger, path string) MachineIPMISuperUser {
	password := ""

	if raw, err := os.ReadFile(path); err == nil {
		password = strings.TrimSpace(string(raw))
		if password != "" {
			log.Info("ipmi superuser password found, feature is enabled")
		} else {
			log.Warn("ipmi superuser password file found, but password is empty, feature is disabled")
		}
	} else {
		log.Warn("ipmi superuser password could not be read, feature is disabled", "error", err)
	}

	return MachineIPMISuperUser{
		password: password,
	}
}

func (i *MachineIPMISuperUser) IsEnabled() bool {
	return i.password != ""
}

func (i *MachineIPMISuperUser) Password() string {
	return i.password
}

func (i *MachineIPMISuperUser) User() string {
	return "root"
}

func DisabledIPMISuperUser() MachineIPMISuperUser {
	return MachineIPMISuperUser{}
}

func (d DNSServers) Validate() error {
	if d == nil {
		return nil
	}

	if len(d) > 3 {
		return errors.New("please specify a maximum of three dns servers")
	}

	for _, dnsServer := range d {
		_, err := netip.ParseAddr(dnsServer.IP)
		if err != nil {
			return fmt.Errorf("ip: %s for dns server not correct err: %w", dnsServer, err)
		}
	}
	return nil
}

func (n NTPServers) Validate() error {
	if n == nil {
		return nil
	}

	if len(n) < 3 || len(n) > 5 {
		return errors.New("please specify a minimum of 3 and a maximum of 5 ntp servers")
	}

	for _, ntpserver := range n {
		if net.ParseIP(ntpserver.Address) != nil {
			_, err := netip.ParseAddr(ntpserver.Address)
			if err != nil {
				return fmt.Errorf("ip: %s for ntp server not correct err: %w", ntpserver, err)
			}
		} else {
			if !govalidator.IsDNSName(ntpserver.Address) {
				return fmt.Errorf("dns name: %s for ntp server not correct", ntpserver)
			}
		}
	}
	return nil
}
