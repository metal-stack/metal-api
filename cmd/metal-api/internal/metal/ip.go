package metal

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// IPType is the type of an ip.
type IPType string

type IPScope string

const (
	// TagIpMachineId is used to tag ips for the usage by machines
	TagIPMachineID = "metal.metal-pod.io/machineid"
	// TagIpClusterId is used to tag ips for the usage for cluster services
	TagIPClusterID = "cluster.metal-pod.io/clusterid"

	// Ephemeral IPs will be cleaned up automatically on machine, network, project deletion
	Ephemeral IPType = "ephemeral"
	// Static IPs will not be cleaned up and can be re-used for machines, networks within a project
	Static IPType = "static"
	// ScopeProject IPs can be assigned to machines or used by cluster services
	ScopeProject IPScope = "project"
	// ScopeMachine IPs are bound to the usage directly at machines
	ScopeMachine IPScope = "machine"
	// ScopeCluster IPs are bound to the usage for cluster services
	ScopeCluster IPScope = "cluster"
)

// IP of a machine/firewall.
type IP struct {
	IPAddress        string    `rethinkdb:"id"`
	ParentPrefixCidr string    `rethinkdb:"prefix"`
	Name             string    `rethinkdb:"name"`
	Description      string    `rethinkdb:"description"`
	ProjectID        string    `rethinkdb:"projectid"`
	NetworkID        string    `rethinkdb:"networkid"`
	Type             IPType    `rethinkdb:"type"`
	Tags             []string  `rethinkdb:"tags"`
	Created          time.Time `rethinkdb:"created"`
	Changed          time.Time `rethinkdb:"changed"`
}

// GetID returns the ID of the entity
func (ip *IP) GetID() string {
	return ip.IPAddress
}

// SetID sets the ID of the entity
func (ip *IP) SetID(id string) {
	ip.IPAddress = id
}

// GetChanged returns the last changed timestamp of the entity
func (ip *IP) GetChanged() time.Time {
	return ip.Changed
}

// SetChanged sets the last changed timestamp of the entity
func (ip *IP) SetChanged(changed time.Time) {
	ip.Changed = changed
}

// GetCreated returns the creation timestamp of the entity
func (ip *IP) GetCreated() time.Time {
	return ip.Created
}

// SetCreated sets the creation timestamp of the entity
func (ip *IP) SetCreated(created time.Time) {
	ip.Created = created
}

// ASNBase is the offset for all Machine ASN´s
const ASNBase = int64(4200000000)

// ASN calculate a ASN from the ip
// we start to calculate ASNs for machines with the first ASN in the 32bit ASN range and
// add the last 2 octets of the ip of the machine to achieve unique ASNs per vrf
// TODO consider using IntegerPool here as well to calculate the addition to ASNBase
func (ip *IP) ASN() (int64, error) {

	i := net.ParseIP(ip.IPAddress)
	if i == nil {
		return int64(-1), errors.Errorf("unable to parse ip %s", ip.IPAddress)
	}
	asn := ASNBase + int64(i[14])*256 + int64(i[15])
	return asn, nil
}

func (ip *IP) GetScope() IPScope {
	for _, t := range ip.Tags {
		if strings.HasPrefix(t, TagIPMachineID) {
			return ScopeMachine
		}
		if strings.HasPrefix(t, TagIPClusterID) {
			return ScopeCluster
		}
	}
	return ScopeProject
}

func (ip *IP) HasMachineId(id string) bool {
	return ip.hasTag(fmt.Sprintf("%s=%s", TagIPMachineID, id))
}

func (ip *IP) AddMachineId(id string) {
	r := []string{}
	for _, t := range ip.Tags {
		if t == TagIPMachineID {
			continue
		}
		r = append(r, t)
	}
	ip.Tags = append(r, fmt.Sprintf("%s=%s", TagIPMachineID, id))
}

func (ip *IP) RemoveMachineId(id string) {
	ip.clearTag(fmt.Sprintf("%s=%s", TagIPMachineID, id))
}

func (ip *IP) hasTag(tag string) bool {
	for _, t := range ip.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

func (ip *IP) clearTag(tag string) {
	r := []string{}
	for _, t := range ip.Tags {
		if t == tag {
			s := strings.Split(t, "=")
			r = append(r, s[0])
			continue
		}
		r = append(r, t)
	}
	ip.Tags = r
}

type IPs []IP

type IPsMap map[string]IPs

func (l IPs) ByProjectID() IPsMap {
	res := IPsMap{}
	for _, e := range l {
		res[e.ProjectID] = append(res[e.ProjectID], e)
	}
	return res
}
