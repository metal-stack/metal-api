package datastore

import (
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
)

// FindIP returns an ip of a given id.
func (rs *RethinkStore) FindIP(id string) (*metal.IP, error) {
	var ip metal.IP
	err := rs.findEntityByID(rs.ipTable(), &ip, id)
	return &ip, err
}

// ListIPs returns all ips.
func (rs *RethinkStore) ListIPs() ([]metal.IP, error) {
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
	return rs.deleteEntityByID(rs.ipTable(), ip.GetID())
}

// UpdateIP updates an ip.
func (rs *RethinkStore) UpdateIP(oldIP *metal.IP, newIP *metal.IP) error {
	return rs.updateEntity(rs.ipTable(), newIP, oldIP)
}
