package metal

import (
	"fmt"
	"net"
	"net/netip"
	"strconv"
	"strings"
)

// A MacAddress is the type for mac adresses. When using a
// custom type, we cannot use strings directly.
type MacAddress string

// Nic information.
type Nic struct {
	MacAddress MacAddress `rethinkdb:"macAddress" json:"macAddress"`
	Name       string     `rethinkdb:"name" json:"name"`
	Identifier string     `rethinkdb:"identifier" json:"identifier"`
	Vrf        string     `rethinkdb:"vrf" json:"vrf"`
	Neighbors  Nics       `rethinkdb:"neighbors" json:"neighbors"`
	Hostname   string     `rethinkdb:"hostname" json:"hostname"`
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
type Prefix struct {
	IP     string `rethinkdb:"ip" json:"ip"`
	Length string `rethinkdb:"length" json:"length"`
}

// Prefixes is an array of prefixes
type Prefixes []Prefix

// NewPrefixFromCIDR returns a new prefix from a given cidr.
func NewPrefixFromCIDR(cidr string) (*Prefix, error) {
	p, err := netip.ParsePrefix(cidr)
	if err != nil {
		return nil, fmt.Errorf("cannot parse prefix from cidr %w", err)
	}
	ip := p.Addr().String()
	length := fmt.Sprintf("%d", p.Bits())
	return &Prefix{
		IP:     ip,
		Length: length,
	}, nil
}

// SplitCIDR return ip and optional length of a prefix
// TODO this is kinda duplicate of NewPrefixFromCIDR
// TODO use net/netip helpers
func SplitCIDR(cidr string) (string, *int) {
	ip, bits, ok := strings.Cut(cidr, "/")
	if ok {
		length, err := strconv.Atoi(bits)
		if err != nil {
			return ip, nil
		}
		return ip, &length
	}

	return cidr, nil
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

// Equals returns true when prefixes have the same cidr.
func (p *Prefix) Equals(other *Prefix) bool {
	return p.String() == other.String()
}

// Network is a network in a metal as a service infrastructure.
// TODO specify rethinkdb restrictions.
type Network struct {
	Base
	Prefixes            Prefixes          `rethinkdb:"prefixes" json:"prefixes"`
	DestinationPrefixes Prefixes          `rethinkdb:"destinationprefixes" json:"destinationprefixes"`
	PartitionID         string            `rethinkdb:"partitionid" json:"partitionid"`
	ProjectID           string            `rethinkdb:"projectid" json:"projectid"`
	ParentNetworkID     string            `rethinkdb:"parentnetworkid" json:"parentnetworkid"`
	Vrf                 uint              `rethinkdb:"vrf" json:"vrf"`
	PrivateSuper        bool              `rethinkdb:"privatesuper" json:"privatesuper"`
	Nat                 bool              `rethinkdb:"nat" json:"nat"`
	Underlay            bool              `rethinkdb:"underlay" json:"underlay"`
	Shared              bool              `rethinkdb:"shared" json:"shared"`
	Labels              map[string]string `rethinkdb:"labels" json:"labels"`
}

// Networks is a list of networks.
type Networks []Network

// NetworkMap is an indexed map of networks
type NetworkMap map[string]Network

// NetworkUsage contains usage information of a network
type NetworkUsage struct {
	AvailableIPs      uint64 `json:"available_ips" description:"the total available IPs" readonly:"true"`
	UsedIPs           uint64 `json:"used_ips" description:"the total used IPs" readonly:"true"`
	AvailablePrefixes uint64 `json:"available_prefixes" description:"the total available Prefixes" readonly:"true"`
	UsedPrefixes      uint64 `json:"used_prefixes" description:"the total used Prefixes" readonly:"true"`
}

// ByID creates an indexed map of partitions whre the id is the index.
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

// SubstractPrefixes returns the prefixes of the network minus the prefixes passed in the arguments
func (n *Network) SubstractPrefixes(prefixes ...Prefix) []Prefix {
	var result []Prefix
	for _, p := range n.Prefixes {
		contains := false
		for i := range prefixes {
			if p.Equals(&prefixes[i]) {
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

// ByName creates a map (nic names --> nic) from a nic list.
func (nics Nics) ByName() map[string]*Nic {
	res := make(map[string]*Nic)

	for i, n := range nics {
		res[n.Name] = &nics[i]
	}

	return res
}

// ByIdentifier creates a map (nic identifier --> nic) from a nic list.
func (nics Nics) ByIdentifier() map[string]*Nic {
	res := make(map[string]*Nic)

	for i, n := range nics {
		res[n.GetIdentifier()] = &nics[i]
	}

	return res
}
