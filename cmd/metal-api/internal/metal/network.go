package metal

import (
	"fmt"
	"net"
	"net/netip"
	"slices"
	"strconv"
	"strings"

	"github.com/samber/lo"
)

// SwitchPortStatus is a type alias for a string that represents the status of a switch port.
// Valid values are defined as constants in this package.
type SwitchPortStatus string

// SwitchPortStatus defines the possible statuses for a switch port.
// UNKNOWN indicates the status is not known.
// UP indicates the port is up and operational.
// DOWN indicates the port is down and not operational.
const (
	SwitchPortStatusUnknown SwitchPortStatus = "UNKNOWN"
	SwitchPortStatusUp      SwitchPortStatus = "UP"
	SwitchPortStatusDown    SwitchPortStatus = "DOWN"
)

// IsConcrete returns true if the SwitchPortStatus is UP or DOWN,
// which are concrete, known statuses. It returns false if the status
// is UNKNOWN, which indicates the status is not known.
func (s SwitchPortStatus) IsConcrete() bool {
	return s == SwitchPortStatusUp || s == SwitchPortStatusDown
}

// IsValid returns true if the SwitchPortStatus is a known valid value
// (UP, DOWN, UNKNOWN).
func (s SwitchPortStatus) IsValid() bool {
	return s == SwitchPortStatusUp || s == SwitchPortStatusDown || s == SwitchPortStatusUnknown
}

// A MacAddress is the type for mac addresses. When using a
// custom type, we cannot use strings directly.
type MacAddress string

// Nic information.
type Nic struct {
	MacAddress   MacAddress          `rethinkdb:"macAddress" json:"macAddress"`
	Name         string              `rethinkdb:"name" json:"name"`
	Identifier   string              `rethinkdb:"identifier" json:"identifier"`
	Vrf          string              `rethinkdb:"vrf" json:"vrf"`
	Neighbors    Nics                `rethinkdb:"neighbors" json:"neighbors"`
	Hostname     string              `rethinkdb:"hostname" json:"hostname"`
	State        *NicState           `rethinkdb:"state" json:"state"`
	BGPPortState *SwitchBGPPortState `rethinkdb:"bgpPortState" json:"bgpPortState"`
}

// NicState represents the desired and actual state of a network interface
// controller (NIC). The Desired field indicates the intended state of the
// NIC, while Actual indicates its current operational state. The Desired
// state will be removed when the actual state is equal to the desired state.
type NicState struct {
	Desired *SwitchPortStatus `rethinkdb:"desired" json:"desired"`
	Actual  SwitchPortStatus  `rethinkdb:"actual" json:"actual"`
}

type SwitchBGPPortState struct {
	Neighbor              string
	PeerGroup             string
	VrfName               string
	BgpState              string
	BgpTimerUpEstablished int64
	SentPrefixCounter     int64
	AcceptedPrefixCounter int64
}

// SetState updates the NicState with the given SwitchPortStatus. It returns
// a new NicState and a bool indicating if the state was changed.
//
// If the given status matches the current Actual state, it checks if Desired
// is set and matches too. If so, Desired is set to nil since the desired
// state has been reached.
//
// If the given status differs from the current Actual state, Desired is left
// unchanged if it differs from the new state so the desired state is still tracked.
// The Actual state is updated to the given status.
//
// This allows tracking both the desired and actual states, while clearing
// Desired once the desired state is achieved.
func (ns *NicState) SetState(s SwitchPortStatus) (NicState, bool) {
	if ns == nil {
		return NicState{
			Actual:  s,
			Desired: nil,
		}, true
	}
	if ns.Actual == s {
		if ns.Desired != nil {
			if *ns.Desired == s {
				// we now have the desired state, so set the desired state to nil
				return NicState{
					Actual:  s,
					Desired: nil,
				}, true
			} else {
				// we already have the reported state, but the desired one is different
				// so nothing changed
				return *ns, false
			}
		}
		// nothing changed
		return *ns, false
	}
	// we got another state as we had before
	if ns.Desired != nil {
		if *ns.Desired == s {
			// we now have the desired state, so set the desired state to nil
			return NicState{
				Actual:  s,
				Desired: nil,
			}, true
		} else {
			// a new state was reported, but the desired one is different
			// so we have to update the state but keep the desired state
			return NicState{
				Actual:  s,
				Desired: ns.Desired,
			}, true
		}
	}
	return NicState{
		Actual:  s,
		Desired: nil,
	}, true
}

// WantState sets the desired state for the NIC. It returns a new NicState
// struct with the desired state set and a bool indicating if the state changed.
// If the current state already matches the desired state, it returns a state
// with a cleared desired field.
func (ns *NicState) WantState(s SwitchPortStatus) (NicState, bool) {
	if ns == nil {
		return NicState{
			Actual:  SwitchPortStatusUnknown,
			Desired: &s,
		}, true
	}
	if ns.Actual == s {
		// we want a state we already have
		if ns.Desired != nil {
			return NicState{
				Actual:  s,
				Desired: nil,
			}, true
		}
		return *ns, false
	}
	// return a new state with the desired state set and a bool indicating a state change
	// only if the desired state is different from the current one
	return NicState{
		Actual:  ns.Actual,
		Desired: &s,
	}, lo.FromPtr(ns.Desired) != s
}

// GetIdentifier returns the identifier of a nic.
// It returns the mac address as a fallback if no identifier was found.
// (this is for backwards compatibility with old metal-core and metal-hammer versions)
func (n *Nic) GetIdentifier() string {
	if n.Identifier != "" {
		return n.Identifier
	}
	return string(n.MacAddress)
}

// Nics is a list of nics.
type Nics []Nic

// Prefix is a ip with mask, either ipv4/ipv6
// FIXME this should be converted to simply a string
type Prefix struct {
	IP     string `rethinkdb:"ip" json:"ip"`
	Length string `rethinkdb:"length" json:"length"`
}

// Prefixes is an array of prefixes
type Prefixes []Prefix

// NewPrefixFromCIDR returns a new prefix from a given cidr.
func NewPrefixFromCIDR(cidr string) (*Prefix, *netip.Prefix, error) {
	prefix, err := netip.ParsePrefix(cidr)
	if err != nil {
		return nil, nil, err
	}
	ip := prefix.Addr().String()
	length := strconv.Itoa(prefix.Bits())
	return &Prefix{
		IP:     ip,
		Length: length,
	}, &prefix, nil
}

// String implements the Stringer interface
func (p *Prefix) String() string {
	return p.IP + "/" + p.Length
}

func (p Prefixes) String() []string {
	result := []string{}
	for _, element := range p {
		result = append(result, element.String())
	}
	return result
}

// OfFamily returns the prefixes of the given address family.
// be aware that malformed prefixes are just skipped, so do not use this for validation or something.
func (p Prefixes) OfFamily(af AddressFamily) Prefixes {
	var res Prefixes

	for _, prefix := range p {
		pfx, err := netip.ParsePrefix(prefix.String())
		if err != nil {
			continue
		}

		if pfx.Addr().Is4() && af == IPv6AddressFamily {
			continue
		}
		if pfx.Addr().Is6() && af == IPv4AddressFamily {
			continue
		}

		res = append(res, prefix)
	}

	return res
}

// AddressFamilies returns the addressfamilies of given prefixes.
// be aware that malformed prefixes are just skipped, so do not use this for validation or something.
func (p Prefixes) AddressFamilies() AddressFamilies {
	var afs AddressFamilies

	for _, prefix := range p {
		pfx, err := netip.ParsePrefix(prefix.String())
		if err != nil {
			continue
		}

		var af AddressFamily
		if pfx.Addr().Is4() {
			af = IPv4AddressFamily
		}
		if pfx.Addr().Is6() {
			af = IPv6AddressFamily
		}
		if !slices.Contains(afs, af) {
			afs = append(afs, af)
		}
	}

	return afs
}

// equals returns true when prefixes have the same cidr.
func (p *Prefix) equals(other *Prefix) bool {
	return p.String() == other.String()
}

// Network is a network in a metal as a service infrastructure.
// TODO specify rethinkdb restrictions.
type Network struct {
	Base
	Prefixes                 Prefixes          `rethinkdb:"prefixes" json:"prefixes"`
	DestinationPrefixes      Prefixes          `rethinkdb:"destinationprefixes" json:"destinationprefixes"`
	DefaultChildPrefixLength ChildPrefixLength `rethinkdb:"defaultchildprefixlength" json:"defaultchildprefixlength" description:"if privatesuper, this defines the bitlen of child prefixes per addressfamily if not nil"`
	PartitionID              string            `rethinkdb:"partitionid" json:"partitionid"`
	ProjectID                string            `rethinkdb:"projectid" json:"projectid"`
	ParentNetworkID          string            `rethinkdb:"parentnetworkid" json:"parentnetworkid"`
	Vrf                      uint              `rethinkdb:"vrf" json:"vrf"`
	PrivateSuper             bool              `rethinkdb:"privatesuper" json:"privatesuper"`
	Nat                      bool              `rethinkdb:"nat" json:"nat"`
	Underlay                 bool              `rethinkdb:"underlay" json:"underlay"`
	Shared                   bool              `rethinkdb:"shared" json:"shared"`
	Labels                   map[string]string `rethinkdb:"labels" json:"labels"`
	// AddressFamilies            AddressFamilies   `rethinkdb:"addressfamilies" json:"addressfamilies"`
	AdditionalAnnouncableCIDRs []string `rethinkdb:"additionalannouncablecidrs" json:"additionalannouncablecidrs" description:"list of cidrs which are added to the route maps per tenant private network, these are typically pod- and service cidrs, can only be set in a supernetwork"`
}

type ChildPrefixLength map[AddressFamily]uint8

// AddressFamily identifies IPv4/IPv6
type AddressFamily string
type AddressFamilies []AddressFamily

const (
	// InvalidAddressFamily identifies a invalid Addressfamily
	InvalidAddressFamily = AddressFamily("invalid")
	// IPv4AddressFamily identifies IPv4
	IPv4AddressFamily = AddressFamily("IPv4")
	// IPv6AddressFamily identifies IPv6
	IPv6AddressFamily = AddressFamily("IPv6")
)

// ToAddressFamily will convert a string af to a AddressFamily
func ToAddressFamily(af string) (AddressFamily, error) {
	switch strings.ToLower(af) {
	case "ipv4":
		return IPv4AddressFamily, nil
	case "ipv6":
		return IPv6AddressFamily, nil
	}
	return InvalidAddressFamily, fmt.Errorf("given addressfamily:%q is invalid", af)
}

// Networks is a list of networks.
type Networks []Network

// NetworkMap is an indexed map of networks
type NetworkMap map[string]Network

// NetworkUsage contains usage information of a network
type NetworkUsage struct {
	AvailableIPs      uint64 `json:"available_ips" description:"the total available IPs" readonly:"true"`
	UsedIPs           uint64 `json:"used_ips" description:"the total used IPs" readonly:"true"`
	AvailablePrefixes uint64 `json:"available_prefixes" description:"the total available 2 bit Prefixes" readonly:"true"`
	UsedPrefixes      uint64 `json:"used_prefixes" description:"the total used Prefixes" readonly:"true"`
}

// ByID creates an indexed map of networks where the id is the index.
func (nws Networks) ByID() NetworkMap {
	res := make(NetworkMap)
	for i, nw := range nws {
		res[nw.ID] = nws[i]
	}
	return res
}

// FindPrefix returns the prefix by cidr if contained in this network, nil otherwise
func (n *Network) FindPrefix(cidr string) *Prefix {
	var found *Prefix
	for _, p := range n.Prefixes {
		if p.String() == cidr {
			return &p
		}
	}
	return found
}

// ContainsIP checks whether the given ip is included in the networks prefixes
func (n *MachineNetwork) ContainsIP(ip string) bool {
	pip := net.ParseIP(ip)
	for _, p := range n.Prefixes {
		_, n, err := net.ParseCIDR(p)
		if err != nil {
			continue
		}
		if n.Contains(pip) {
			return true
		}
	}
	return false
}

// SubtractPrefixes returns the prefixes of the network minus the prefixes passed in the arguments
func (n *Network) SubtractPrefixes(prefixes ...Prefix) []Prefix {
	var result []Prefix
	for _, p := range n.Prefixes {
		contains := false
		for i := range prefixes {
			if p.equals(&prefixes[i]) {
				contains = true
				break
			}
		}
		if contains {
			continue
		}
		result = append(result, p)
	}
	return result
}

func (nics Nics) FilterByHostname(hostname string) (res Nics) {
	if hostname == "" {
		return nics
	}

	for i, n := range nics {
		if n.Hostname == hostname {
			res = append(res, nics[i])
		}
	}

	return res
}

// NicMap maps nic names to the corresponding nics
type NicMap map[string]*Nic

// ByName creates a map (nic names --> nic) from a nic list.
func (nics Nics) ByName() NicMap {
	res := make(NicMap)

	for i, n := range nics {
		res[n.Name] = &nics[i]
	}

	return res
}

// ByIdentifier creates a map (nic identifier --> nic) from a nic list.
func (nics Nics) ByIdentifier() NicMap {
	res := make(NicMap)

	for i, n := range nics {
		res[n.GetIdentifier()] = &nics[i]
	}

	return res
}
