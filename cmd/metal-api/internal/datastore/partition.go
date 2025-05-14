package datastore

import (
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
)

// FindPartition return a partition for the given id.
func (rs *RethinkStore) FindPartition(id string) (*metal.Partition, error) {
	var p metal.Partition
	err := rs.findEntityByID(rs.partitionTable(), &p, id)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// ListPartitions returns all partition.
func (rs *RethinkStore) ListPartitions() (metal.Partitions, error) {
	ps := make(metal.Partitions, 0)
	err := rs.listEntities(rs.partitionTable(), &ps)
	return ps, err
}

// CreatePartition creates a new partition.
func (rs *RethinkStore) CreatePartition(p *metal.Partition) error {
	return rs.createEntity(rs.partitionTable(), p)
}

// DeletePartition deletes a partition.
func (rs *RethinkStore) DeletePartition(p *metal.Partition) error {
	return rs.deleteEntity(rs.partitionTable(), p)
}

// UpdatePartition updates a partition.
func (rs *RethinkStore) UpdatePartition(oldPartition *metal.Partition, newPartition *metal.Partition) error {
	return rs.updateEntity(rs.partitionTable(), newPartition, oldPartition)
}
