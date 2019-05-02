package metal

import (
	"time"
)

// IP of a machine/firewall.
type IP struct {
	IPAddress        string    `json:"ipaddress" description:"the ip address (ipv4 or ipv6) of this ip, required." rethinkdb:"id"`
	ParentPrefixCidr string    `json:"prefix" description:"the prefix cidr in which this ip was created." rethinkdb:"prefix"`
	Name             string    `json:"name" description:"the readable name" optional:"true" rethinkdb:"name"`
	Description      string    `json:"description,omitempty" description:"a description for this entity" optional:"true" rethinkdb:"description"`
	Created          time.Time `json:"created" description:"the creation time of this entity" optional:"true" readOnly:"true" rethinkdb:"created"`
	Changed          time.Time `json:"changed" description:"the last changed timestamp" optional:"true" readOnly:"true" rethinkdb:"changed"`
	NetworkID        string    `json:"networkid" description:"the network this ip address belongs to, required." rethinkdb:"networkid"`
	ProjectID        string    `json:"projectid" description:"the project this ip address belongs to, required." rethinkdb:"projectid"`
}

func (ip *IP) GetID() string {
	return ip.IPAddress
}

func (ip *IP) SetID(id string) {
	ip.IPAddress = id
}

func (ip *IP) GetChanged() time.Time {
	return ip.Changed
}

func (ip *IP) SetChanged(changed time.Time) {
	ip.Changed = changed
}

func (ip *IP) SetCreated(created time.Time) {
	ip.Created = created
}
