package datastore

import (
	"context"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
)

// FindPartition return a partition for the given id.
func (rs *RethinkStore) FindPartition(ctx context.Context, id string) (*metal.Partition, error) {
	var p metal.Partition
	err := rs.findEntityByID(ctx, rs.partitionTable(), &p, id)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// ListPartitions returns all partition.
func (rs *RethinkStore) ListPartitions(ctx context.Context) (metal.Partitions, error) {
	ps := make(metal.Partitions, 0)
	err := rs.listEntities(ctx, rs.partitionTable(), &ps)
	return ps, err
}

// CreatePartition creates a new partition.
func (rs *RethinkStore) CreatePartition(ctx context.Context, p *metal.Partition) error {
	return rs.createEntity(ctx, rs.partitionTable(), p)
}

// DeletePartition delets a partition.
func (rs *RethinkStore) DeletePartition(ctx context.Context, p *metal.Partition) error {
	return rs.deleteEntity(ctx, rs.partitionTable(), p)
}

// UpdatePartition updates a partition.
func (rs *RethinkStore) UpdatePartition(ctx context.Context, oldPartition *metal.Partition, newPartition *metal.Partition) error {
	return rs.updateEntity(ctx, rs.partitionTable(), newPartition, oldPartition)
}
