package datastore

import (
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
	v1 "git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/service/v1"
	r "gopkg.in/rethinkdb/rethinkdb-go.v5"
)

// FindIP returns an ip of a given id.
func (rs *RethinkStore) FindIP(id string) (*metal.IP, error) {
	var ip metal.IP
	err := rs.findEntityByID(rs.ipTable(), &ip, id)
	if err != nil {
		return nil, err
	}
	return &ip, nil
}

// FindIPs returns the ips that match the given properties
func (rs *RethinkStore) FindIPs(props *v1.FindIPsRequest) ([]metal.IP, error) {
	q := *rs.ipTable()

	if props.IPAddress != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("id").Eq(*props.IPAddress)
		})
	}

	if props.ProjectID != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("projectid").Eq(*props.ProjectID)
		})
	}

	if props.NetworkID != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("networkid").Eq(*props.NetworkID)
		})
	}

	if props.ParentPrefixCidr != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("networkprefix").Eq(*props.ParentPrefixCidr)
		})
	}

	if props.MachineID != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("machineid").Eq(*props.MachineID)
		})
	}

	var ips []metal.IP
	err := rs.searchEntities(&q, &ips)
	if err != nil {
		return nil, err
	}

	return ips, nil
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
