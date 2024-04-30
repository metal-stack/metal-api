package metal

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

// A MacAddress is the type for mac addresses. When using a
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
	parts := strings.Split(cidr, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("cannot split cidr into pieces: %v", cidr)
	}
	ip := strings.TrimSpace(parts[0])
	length := strings.TrimSpace(parts[1])
	return &Prefix{
		IP:     ip,
		Length: length,
	}, nil
}

func SplitCIDR(cidr string) (string, *int) {
	parts := strings.Split(cidr, "/")
	if len(parts) == 2 {
		length, err := strconv.Atoi(parts[1])
		if err != nil {
			return parts[0], nil
		}
		return parts[0], &length
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

// ByID creates an indexed map of partitions where the id is the index.
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
