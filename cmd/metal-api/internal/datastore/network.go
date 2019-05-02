package datastore

import (
	"fmt"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
	r "gopkg.in/rethinkdb/rethinkdb-go.v5"
)

// GetPrimaryNetwork returns the network which is marked default
func (rs *RethinkStore) GetPrimaryNetwork() (*metal.Network, error) {
	var nws []metal.Network
	searchFilter := func(row r.Term) r.Term {
		return row.Field("primary").Eq(true)
	}

	err := rs.searchEntities(rs.networkTable(), searchFilter, &nws)
	if err != nil {
		return nil, err
	}
	if len(nws) == 0 {
		return nil, fmt.Errorf("no primary network in the database")
	}
	if len(nws) > 1 {
		return nil, fmt.Errorf("more than one primary network in the database, which should not be the case")
	}

	return &nws[0], nil
}

// SearchProjectNetwork returns the network to a given project id
func (rs *RethinkStore) SearchProjectNetwork(projectid string) (*metal.Network, error) {
	var nws []metal.Network
	searchFilter := func(row r.Term) r.Term {
		return row.Field("projectid").Eq(projectid)
	}

	err := rs.searchEntities(rs.networkTable(), searchFilter, &nws)
	if err != nil {
		return nil, err
	}
	if len(nws) == 0 {
		return nil, nil
	}
	if len(nws) > 1 {
		return nil, fmt.Errorf("found multiple network for project %s, which should never be the case", projectid)
	}

	return &nws[0], nil
}

// FindNetwork returns an network of a given id.
func (rs *RethinkStore) FindNetwork(id string) (*metal.Network, error) {
	var nw metal.Network
	err := rs.findEntityByID(rs.networkTable(), &nw, id)
	return &nw, err
}

// ListNetworks returns all networks.
func (rs *RethinkStore) ListNetworks() ([]metal.Network, error) {
	nws := make([]metal.Network, 0)
	err := rs.listEntities(rs.networkTable(), &nws)
	return nws, err
}

// CreateNetwork creates a new network.
func (rs *RethinkStore) CreateNetwork(nw *metal.Network) error {
	return rs.createEntity(rs.networkTable(), nw)
}

// DeleteNetwork deletes an network.
func (rs *RethinkStore) DeleteNetwork(nw *metal.Network) error {
	return rs.deleteEntityByID(rs.networkTable(), nw.GetID())
}

// UpdateNetwork updates an network.
func (rs *RethinkStore) UpdateNetwork(oldNetwork *metal.Network, newNetwork *metal.Network) error {
	return rs.updateEntity(rs.networkTable(), newNetwork, oldNetwork)
}
