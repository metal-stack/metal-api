package datastore

import (
	"errors"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
)

// FindSize return a size for a given id.
func (rs *RethinkStore) FindSize(id string) (*metal.Size, error) {
	var s metal.Size
	err := rs.findEntityByID(rs.sizeTable(), &s, id)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// ListSizes returns all sizes.
func (rs *RethinkStore) ListSizes() (metal.Sizes, error) {
	szs := make(metal.Sizes, 0)
	err := rs.listEntities(rs.sizeTable(), &szs)
	return szs, err
}

// CreateSize creates a new size.
func (rs *RethinkStore) CreateSize(size *metal.Size) error {
	return rs.createEntity(rs.sizeTable(), size)
}

// DeleteSize deletes a size.
func (rs *RethinkStore) DeleteSize(size *metal.Size) error {
	return rs.deleteEntity(rs.sizeTable(), size)
}

// UpdateSize updates a size.
func (rs *RethinkStore) UpdateSize(oldSize *metal.Size, newSize *metal.Size) error {
	return rs.updateEntity(rs.sizeTable(), newSize, oldSize)
}

// FromHardware tries to find a size which matches the given hardware specs.
func (rs *RethinkStore) FromHardware(hw metal.MachineHardware) (*metal.Size, error) {
	sz, err := rs.ListSizes()
	if err != nil {
		return nil, err
	}
	if len(sz) < 1 {
		// this should not happen, so we do not return a notfound
		return nil, errors.New("no sizes found in database")
	}
	var sizes metal.Sizes
	for _, s := range sz {
		if len(s.Constraints) < 1 {
			rs.log.Error("missing constraints", "size", s)
			continue
		}
		sizes = append(sizes, s)
	}
	return sizes.FromHardware(hw)
}
