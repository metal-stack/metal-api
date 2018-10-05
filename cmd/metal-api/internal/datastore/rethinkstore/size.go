package rethinkstore

import (
	"fmt"

	"git.f-i-ts.de/cloud-native/maas/metal-api/pkg/metal"
	r "gopkg.in/rethinkdb/rethinkdb-go.v5"
)

func (rs *RethinkStore) FindSize(id string) (*metal.Size, error) {
	res, err := rs.sizeTable.Get(id).Run(rs.session)
	if err != nil {
		return nil, fmt.Errorf("cannot get size from database: %v", err)
	}
	defer res.Close()
	var r metal.Size
	err = res.One(&r)
	if err != nil {
		return nil, fmt.Errorf("cannot fetch results: %v", err)
	}
	return &r, nil
}

func (rs *RethinkStore) SearchSize() error {
	return fmt.Errorf("not implemented yet")
}

func (rs *RethinkStore) ListSizes() ([]metal.Size, error) {
	res, err := rs.sizeTable.Run(rs.session)
	if err != nil {
		return nil, fmt.Errorf("cannot search sizes from database: %v", err)
	}
	defer res.Close()
	data := make([]metal.Size, 0)
	err = res.All(&data)
	if err != nil {
		return nil, fmt.Errorf("cannot fetch results: %v", err)
	}
	return data, nil
}

func (rs *RethinkStore) CreateSize(size *metal.Size) error {
	res, err := rs.sizeTable.Insert(size).RunWrite(rs.session)
	if err != nil {
		return fmt.Errorf("cannot create size in database: %v", err)
	}
	if size.ID == "" {
		size.ID = res.GeneratedKeys[0]
	}
	return nil
}

func (rs *RethinkStore) DeleteSize(id string) (*metal.Size, error) {
	sz, err := rs.FindSize(id)
	if err != nil {
		return nil, fmt.Errorf("cannot find size with id %q: %v", id, err)
	}
	_, err = rs.sizeTable.Get(id).Delete().RunWrite(rs.session)
	if err != nil {
		return nil, fmt.Errorf("cannot delete size from database: %v", err)
	}
	return sz, nil
}

func (rs *RethinkStore) DeleteSizes() error {
	// we do not support this here! do we really need such a method?
	return nil
}

func (rs *RethinkStore) UpdateSize(oldSize *metal.Size, newSize *metal.Size) error {
	_, err := rs.sizeTable.Get(oldSize.ID).Replace(func(row r.Term) r.Term {
		return r.Branch(row.Field("changed").Eq(r.Expr(oldSize.Changed)), newSize, r.Error("the size was changed from another, please retry"))
	}).RunWrite(rs.session)
	if err != nil {
		return fmt.Errorf("cannot update size: %v", err)
	}
	return nil
}

func (rs *RethinkStore) determineSizeFromHardware(hardware metal.DeviceHardware) *metal.Size {
	// FIXME: For POC always return small if present
	size, err := rs.FindSize("t1.small.x86")
	if err != nil {
		rs.Error("size could not be determined from the hardware")
		return nil
	}
	return size
}
