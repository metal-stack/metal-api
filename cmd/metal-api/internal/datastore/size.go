package datastore

import (
	"fmt"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
)

// FindSize return a size for a given id.
func (rs *RethinkStore) FindSize(id string) (*metal.Size, error) {
	var s metal.Size
	err := rs.findEntityByID(rs.sizeTable(), &s, id)
	return &s, err
}

// ListSizes returns all sizes.
func (rs *RethinkStore) ListSizes() (metal.Sizes, error) {
	szs := make([]metal.Size, 0)
	err := rs.listEntities(rs.sizeTable(), &szs)
	return szs, err
}

// CreateSize creates a new size.
func (rs *RethinkStore) CreateSize(size *metal.Size) error {
	return rs.createEntity(rs.sizeTable(), size)
}

// DeleteSize deletes a size.
func (rs *RethinkStore) DeleteSize(size *metal.Size) error {
	return rs.deleteEntityByID(rs.sizeTable(), size.GetID())
}

// UpdateSize updates a size.
func (rs *RethinkStore) UpdateSize(oldSize *metal.Size, newSize *metal.Size) error {
	return rs.updateEntity(rs.sizeTable(), newSize, oldSize)
}

// FromHardware tries to find a size which matches the given hardware specs.
func (rs *RethinkStore) FromHardware(hw metal.MachineHardware) (*metal.Size, []*metal.SizeMatchingLog, error) {
	sz, err := rs.ListSizes()
	if err != nil {
		return nil, nil, err
	}
	if len(sz) < 1 {
		// this should not happen, so we do not return a notfound
		return nil, nil, fmt.Errorf("no sizes found in database")
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
