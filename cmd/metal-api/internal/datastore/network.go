package datastore

import (
	"fmt"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
	r "gopkg.in/rethinkdb/rethinkdb-go.v5"
)

// FindPrimaryNetwork returns the network which is marked default in this partition
func (rs *RethinkStore) FindPrimaryNetwork(partitionID string) (*metal.Network, error) {
	_, err := rs.FindPartition(partitionID)
	if err != nil {
		return nil, err
	}

	var nws []metal.Network
	searchFilter := func(row r.Term) r.Term {
		return row.Field("primary").Eq(true).And(row.Field("partitionid").Eq(partitionID))
	}

	err = rs.searchEntities(rs.networkTable(), searchFilter, &nws)
	if err != nil {
		return nil, err
	}
	if len(nws) == 0 {
		return nil, fmt.Errorf("no primary network in the database in partition:%s found", partitionID)
	}
	if len(nws) > 1 {
		return nil, fmt.Errorf("more than one primary network in partition %s in the database, which should not be the case", partitionID)
	}

	return &nws[0], nil
}

// SearchPrimaryNetwork returns all primary networks of a partition.
func (rs *RethinkStore) SearchPrimaryNetwork(partitionID string) ([]metal.Network, error) {
	_, err := rs.FindPartition(partitionID)
	if err != nil {
		return nil, err
	}

	var nws []metal.Network
	searchFilter := func(row r.Term) r.Term {
		return row.Field("primary").Eq(true).And(row.Field("partitionid").Eq(partitionID))
	}

	err = rs.searchEntities(rs.networkTable(), searchFilter, &nws)
	if err != nil {
		return nil, err
	}

	return nws, nil
}

// SearchUnderlayNetwork returns the network which is marked as underlay in this partition
func (rs *RethinkStore) SearchUnderlayNetwork(partitionID string) ([]metal.Network, error) {
	_, err := rs.FindPartition(partitionID)
	if err != nil {
		return nil, err
	}

	var nws []metal.Network
	searchFilter := func(row r.Term) r.Term {
		return row.Field("underlay").Eq(true).And(row.Field("partitionid").Eq(partitionID))
	}

	err = rs.searchEntities(rs.networkTable(), searchFilter, &nws)
	if err != nil {
		return nil, err
	}

	return nws, nil
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
	if err != nil {
		return nil, err
	}
	return &nw, nil
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
	return rs.deleteEntity(rs.networkTable(), nw)
}

// UpdateNetwork updates an network.
func (rs *RethinkStore) UpdateNetwork(oldNetwork *metal.Network, newNetwork *metal.Network) error {
	return rs.updateEntity(rs.networkTable(), newNetwork, oldNetwork)
}
