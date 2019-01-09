package datastore

import (
	"fmt"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
	r "gopkg.in/rethinkdb/rethinkdb-go.v5"
)

// FindSize return a size for a given id.
func (rs *RethinkStore) FindSize(id string) (*metal.Size, error) {
	res, err := rs.table("size").Get(id).Run(rs.session)
	if err != nil {
		return nil, fmt.Errorf("cannot get size from database: %v", err)
	}
	defer res.Close()
	if res.IsNil() {
		return nil, metal.NotFound("no size %q found", id)
	}
	var r metal.Size
	err = res.One(&r)
	if err != nil {
		return nil, fmt.Errorf("cannot fetch results: %v", err)
	}
	return &r, nil
}

// ListSizes returns all sizes.
func (rs *RethinkStore) ListSizes() ([]metal.Size, error) {
	res, err := rs.table("size").Run(rs.session)
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

// CreateSize creates a new size.
func (rs *RethinkStore) CreateSize(size *metal.Size) error {
	res, err := rs.table("size").Insert(size).RunWrite(rs.session)
	if err != nil {
		return fmt.Errorf("cannot create size in database: %v", err)
	}
	if size.ID == "" {
		size.ID = res.GeneratedKeys[0]
	}
	return nil
}

// DeleteSize deletes a size.
func (rs *RethinkStore) DeleteSize(id string) (*metal.Size, error) {
	sz, err := rs.FindSize(id)
	if err != nil {
		return nil, err
	}
	_, err = rs.table("size").Get(id).Delete().RunWrite(rs.session)
	if err != nil {
		return nil, fmt.Errorf("cannot delete size from database: %v", err)
	}
	return sz, nil
}

// UpdateSize updates a size.
func (rs *RethinkStore) UpdateSize(oldSize *metal.Size, newSize *metal.Size) error {
	_, err := rs.table("size").Get(oldSize.ID).Replace(func(row r.Term) r.Term {
		return r.Branch(row.Field("changed").Eq(r.Expr(oldSize.Changed)), newSize, r.Error("the size was changed from another, please retry"))
	}).RunWrite(rs.session)
	if err != nil {
		return fmt.Errorf("cannot update size: %v", err)
	}
	return nil
}

// FromHardware tries to find a size which matches the given hardware specs.
func (rs *RethinkStore) FromHardware(hw metal.DeviceHardware) (*metal.Size, error) {
	sz, err := rs.ListSizes()
	if err != nil {
		return nil, err
	}
	if len(sz) < 1 {
		// this should not happen, so we do not return a notfound
		return nil, fmt.Errorf("no sizes found in database")
	}
	var sizes []metal.Size
	for _, s := range sz {
		if len(s.Constraints) < 1 {
			rs.Error("missing constraints", "size", s)
			continue
		}
		sizes = append(sizes, s)
	}
	return metal.Sizes(sizes).FromHardware(hw)
}
