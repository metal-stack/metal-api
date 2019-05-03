package metal

import (
	"time"
)

// IP of a machine/firewall.
type IP struct {
	IPAddress        string    `modelDescription:"an ip address that can be attached to a machine" rethinkdb:"id"`
	ParentPrefixCidr string    `rethinkdb:"prefix"`
	Name             string    `rethinkdb:"name"`
	Description      string    `rethinkdb:"description"`
	Created          time.Time `rethinkdb:"created"`
	Changed          time.Time `rethinkdb:"changed"`
	NetworkID        string    `rethinkdb:"networkid"`
	ProjectID        string    `rethinkdb:"projectid"`
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
