package datastore

import (
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"github.com/metal-stack/metal-lib/pkg/tag"
	r "gopkg.in/rethinkdb/rethinkdb-go.v6"
)

// IPSearchQuery can be used to search networks.
type IPSearchQuery struct {
	IPAddress        *string  `json:"ipaddress" modelDescription:"an ip address that can be attached to a machine" description:"the address (ipv4 or ipv6) of this ip" optional:"true"`
	ParentPrefixCidr *string  `json:"networkprefix" description:"the prefix of the network this ip address belongs to" optional:"true"`
	NetworkID        *string  `json:"networkid" description:"the network this ip allocate request address belongs to" optional:"true"`
	Tags             []string `json:"tags" description:"the tags that are assigned to this ip address" optional:"true"`
	ProjectID        *string  `json:"projectid" description:"the project this ip address belongs to, empty if not strong coupled" optional:"true"`
	Type             *string  `json:"type" description:"the type of the ip address, ephemeral or static" optional:"true"`
	MachineID        *string  `json:"machineid" description:"the machine an ip address is associated to" optional:"true"`
}

// GenerateTerm generates the project search query term.
func (p *IPSearchQuery) generateTerm(rs *RethinkStore) *r.Term {
	q := *rs.ipTable()

	if p.IPAddress != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("id").Eq(*p.IPAddress)
		})
	}

	if p.ProjectID != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("projectid").Eq(*p.ProjectID)
		})
	}

	if p.NetworkID != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("networkid").Eq(*p.NetworkID)
		})
	}

	if p.ParentPrefixCidr != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("networkprefix").Eq(*p.ParentPrefixCidr)
		})
	}

	if p.MachineID != nil {
		p.Tags = append(p.Tags, metal.IpTag(tag.MachineID, *p.MachineID))
	}

	for _, tag := range p.Tags {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("tags").Contains(r.Expr(tag))
		})
	}

	if p.Type != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("type").Eq(*p.Type)
		})
	}

	return &q
}

// FindIPByID returns an ip of a given id.
func (rs *RethinkStore) FindIPByID(id string) (*metal.IP, error) {
	var ip metal.IP
	err := rs.findEntityByID(rs.ipTable(), &ip, id)
	if err != nil {
		return nil, err
	}
	return &ip, nil
}

// SearchIPs returns the result of the ips search request query.
func (rs *RethinkStore) SearchIPs(q *IPSearchQuery, ips *metal.IPs) error {
	return rs.searchEntities(q.generateTerm(rs), ips)
}

// ListIPs returns all ips.
func (rs *RethinkStore) ListIPs() (metal.IPs, error) {
	ips := make([]metal.IP, 0)
	err := rs.listEntities(rs.ipTable(), &ips)
	return ips, err
}

// CreateIP creates a new ip.
func (rs *RethinkStore) CreateIP(ip *metal.IP) error {
	return rs.createEntity(rs.ipTable(), ip)
}

// DeleteIP deletes an ip.
func (rs *RethinkStore) DeleteIP(ip *metal.IP) error {
	return rs.deleteEntity(rs.ipTable(), ip)
}

// UpdateIP updates an ip.
func (rs *RethinkStore) UpdateIP(oldIP *metal.IP, newIP *metal.IP) error {
	return rs.updateEntity(rs.ipTable(), newIP, oldIP)
}
