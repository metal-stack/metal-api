package datastore

import (
	"fmt"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
	r "gopkg.in/rethinkdb/rethinkdb-go.v5"
)

// FindPartition return a partition for the given id.
func (rs *RethinkStore) FindPartition(id string) (*metal.Partition, error) {
	res, err := rs.partitionTable().Get(id).Run(rs.session)
	if err != nil {
		return nil, fmt.Errorf("cannot get Partition from database: %v", err)
	}
	defer res.Close()
	if res.IsNil() {
		return nil, metal.NotFound("no Partition %q found", id)
	}
	var r metal.Partition
	err = res.One(&r)
	if err != nil {
		return nil, fmt.Errorf("cannot fetch results: %v", err)
	}
	return &r, nil
}

// ListPartitions returns all partition.
func (rs *RethinkStore) ListPartitions() ([]metal.Partition, error) {
	res, err := rs.partitionTable().Run(rs.session)
	if err != nil {
		return nil, fmt.Errorf("cannot search partitions from database: %v", err)
	}
	defer res.Close()
	data := make([]metal.Partition, 0)
	err = res.All(&data)
	if err != nil {
		return nil, fmt.Errorf("cannot fetch results: %v", err)
	}
	return data, nil
}

// CreatePartition creates a new partition.
func (rs *RethinkStore) CreatePartition(f *metal.Partition) (*metal.Partition, error) {
	res, err := rs.partitionTable().Insert(f).RunWrite(rs.session)
	if err != nil {
		return nil, fmt.Errorf("cannot create Partition in database: %v", err)
	}
	if f.ID == "" {
		f.ID = res.GeneratedKeys[0]
	}
	return f, nil
}

// DeletePartition delets a partition.
func (rs *RethinkStore) DeletePartition(id string) (*metal.Partition, error) {
	f, err := rs.FindPartition(id)
	if err != nil {
		return nil, err
	}
	_, err = rs.partitionTable().Get(id).Delete().RunWrite(rs.session)
	if err != nil {
		return nil, fmt.Errorf("cannot delete Partition from database: %v", err)
	}
	return f, nil
}

// UpdatePartition updates a partition.
func (rs *RethinkStore) UpdatePartition(oldF *metal.Partition, newF *metal.Partition) error {
	_, err := rs.partitionTable().Get(oldF.ID).Replace(func(row r.Term) r.Term {
		return r.Branch(row.Field("changed").Eq(r.Expr(oldF.Changed)), newF, r.Error("the Partition was changed from another, please retry"))
	}).RunWrite(rs.session)
	if err != nil {
		return fmt.Errorf("cannot update Partition: %v", err)
	}
	return nil
}
