package metal

import (
	"fmt"
	"strings"
	"time"

	"github.com/metal-stack/metal-lib/pkg/tag"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/tags"
)

// IPType is the type of an ip.
type IPType string

// IPScope is the scope of an ip.
type IPScope string

const (
	// TagIPSeparator is the separator character for key and values in IP-Tags
	TagIPSeparator = "="

	// Ephemeral IPs will be cleaned up automatically on machine, network, project deletion
	Ephemeral IPType = "ephemeral"
	// Static IPs will not be cleaned up and can be re-used for machines, networks within a project
	Static IPType = "static"

	// ScopeEmpty IPs are not bound to a project, machine or cluster
	ScopeEmpty IPScope = ""
	// ScopeProject IPs can be assigned to machines or used by cluster services
	ScopeProject IPScope = "project"
	// ScopeMachine IPs are bound to the usage directly at machines
	ScopeMachine IPScope = "machine"
)

// IP of a machine/firewall.
type IP struct {
	IPAddress string `rethinkdb:"id" json:"id"`
	// AllocationID will be randomly generated during IP creation and helps identifying the point in time
	// when an IP was created. This is not the primary key!
	// This field can help to distinguish whether an IP address was re-acquired or
	// if it is still the same ip address as before.
	AllocationUUID   string    `rethinkdb:"allocationuuid" json:"allocationuuid"`
	ParentPrefixCidr string    `rethinkdb:"prefix" json:"prefix"`
	Name             string    `rethinkdb:"name" json:"name"`
	Description      string    `rethinkdb:"description" json:"description"`
	ProjectID        string    `rethinkdb:"projectid" json:"projectid"`
	NetworkID        string    `rethinkdb:"networkid" json:"networkid"`
	Type             IPType    `rethinkdb:"type" json:"type"`
	Tags             []string  `rethinkdb:"tags" json:"tags"`
	Created          time.Time `rethinkdb:"created" json:"created"`
	Changed          time.Time `rethinkdb:"changed" json:"changed"`
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

// GetScope determines the scope of an ip address
func (ip *IP) GetScope() IPScope {
	if ip.ProjectID == "" {
		return ScopeEmpty
	}
	for _, t := range ip.Tags {
		if strings.HasPrefix(t, tag.MachineID) {
			return ScopeMachine
		}
	}
	return ScopeProject
}

func (ip *IP) HasMachineId(id string) bool {
	t := tags.New(ip.Tags)
	return t.Has(IpTag(tag.MachineID, id))
}

func (ip *IP) GetMachineIds() []string {
	ts := tags.New(ip.Tags)
	return ts.Values(tag.MachineID + TagIPSeparator)
}

func (ip *IP) AddMachineId(id string) {
	ts := tags.New(ip.Tags)
	t := IpTag(tag.MachineID, id)
	ts.Remove(tag.MachineID)
	ts.Add(t)
	ip.Tags = ts.Unique()
}

func (ip *IP) RemoveMachineId(id string) {
	ts := tags.New(ip.Tags)
	t := IpTag(tag.MachineID, id)
	ts.Remove(t)
	ip.Tags = ts.Unique()
}

func IpTag(key, value string) string {
	return fmt.Sprintf("%s%s%s", key, TagIPSeparator, value)
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
