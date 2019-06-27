package datastore

import (
	"fmt"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
	v1 "git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/service/v1"
	r "gopkg.in/rethinkdb/rethinkdb-go.v5"
)

// FindPrimaryNetwork returns the network which is marked default in this partition
func (rs *RethinkStore) FindPrimaryNetwork(partitionID string) (*metal.Network, error) {
	_, err := rs.FindPartition(partitionID)
	if err != nil {
		return nil, err
	}

	q := *rs.networkTable()
	q = q.Filter(func(row r.Term) r.Term {
		return row.Field("primary").Eq(true).And(row.Field("partitionid").Eq(partitionID))
	})

	var nws []metal.Network
	err = rs.searchEntities(&q, &nws)
	if err != nil {
		return nil, err
	}
	if len(nws) == 0 {
		return nil, metal.NotFound("no primary network in the database in partition:%s found", partitionID)
	}
	if len(nws) > 1 {
		return nil, fmt.Errorf("more than one primary network in partition %s in the database, which should not be the case", partitionID)
	}

	return &nws[0], nil
}

// FindPrimaryNetworks returns all primary networks of a partition.
func (rs *RethinkStore) FindPrimaryNetworks(partitionID string) ([]metal.Network, error) {
	_, err := rs.FindPartition(partitionID)
	if err != nil {
		return nil, err
	}

	q := *rs.networkTable()
	q = q.Filter(func(row r.Term) r.Term {
		return row.Field("primary").Eq(true).And(row.Field("partitionid").Eq(partitionID))
	})

	var nws []metal.Network
	err = rs.searchEntities(&q, &nws)
	if err != nil {
		return nil, err
	}

	return nws, nil
}

// FindUnderlayNetworks returns the networks that are marked as underlay in this partition
func (rs *RethinkStore) FindUnderlayNetworks(partitionID string) ([]metal.Network, error) {
	_, err := rs.FindPartition(partitionID)
	if err != nil {
		return nil, err
	}

	q := *rs.networkTable()
	q = q.Filter(func(row r.Term) r.Term {
		return row.Field("underlay").Eq(true).And(row.Field("partitionid").Eq(partitionID))
	})

	var nws []metal.Network
	err = rs.searchEntities(&q, &nws)
	if err != nil {
		return nil, err
	}

	return nws, nil
}

// FindNetworks returns the networks that match the given properties
func (rs *RethinkStore) FindNetworks(props *v1.FindNetworksRequest) ([]metal.Network, error) {
	q := *rs.networkTable()

	if props.ID != nil && *props.ID != "" {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("id").Eq(*props.ID)
		})
	}

	if props.TenantID != nil && *props.TenantID != "" {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("tenantid").Eq(*props.TenantID)
		})
	}

	if props.ProjectID != nil && *props.ProjectID != "" {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("projectid").Eq(*props.ProjectID)
		})
	}

	if props.PartitionID != nil && *props.PartitionID != "" {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("partitionid").Eq(*props.PartitionID)
		})
	}

	if props.ParentNetworkID != nil && *props.ParentNetworkID != "" {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("parentnetworkid").Eq(*props.ParentNetworkID)
		})
	}

	if props.Name != nil && *props.Name != "" {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("name").Eq(*props.Name)
		})
	}

	if props.Vrf != nil && *props.Vrf != 0 {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("vfr").Eq(*props.Vrf)
		})
	}

	if props.Nat != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("nat").Eq(*props.Nat)
		})
	}

	if props.Primary != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("primary").Eq(*props.Primary)
		})
	}

	if props.Underlay != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("underlay").Eq(*props.Underlay)
		})
	}

	for _, prefix := range props.Prefixes {
		if prefix == "" {
			continue
		}
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("prefixes").Slice().Contains(r.Expr(prefix))
		})
	}

	for _, destPrefix := range props.DestinationPrefixes {
		if destPrefix == "" {
			continue
		}
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("destinationprefixes").Slice().Contains(r.Expr(destPrefix))
		})
	}

	var nws []metal.Network
	err := rs.searchEntities(&q, &nws)
	if err != nil {
		return nil, err
	}

	return nws, nil
}

// FindProjectNetwork returns the network to a given project id
func (rs *RethinkStore) FindProjectNetwork(projectid string) (*metal.Network, error) {
	q := *rs.networkTable()
	q = q.Filter(func(row r.Term) r.Term {
		return row.Field("projectid").Eq(projectid)
	})

	var nws []metal.Network
	err := rs.searchEntities(&q, &nws)
	if err != nil {
		return nil, err
	}
	if len(nws) == 0 {
		return nil, metal.NotFound("did not find a project network for project: %s", projectid)
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
