package datastore

import (
	"context"
	"errors"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
)

// FindSize return a size for a given id.
func (rs *RethinkStore) FindSize(ctx context.Context, id string) (*metal.Size, error) {
	var s metal.Size
	err := rs.findEntityByID(ctx, rs.sizeTable(), &s, id)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// ListSizes returns all sizes.
func (rs *RethinkStore) ListSizes(ctx context.Context) (metal.Sizes, error) {
	szs := make(metal.Sizes, 0)
	err := rs.listEntities(ctx, rs.sizeTable(), &szs)
	return szs, err
}

// CreateSize creates a new size.
func (rs *RethinkStore) CreateSize(ctx context.Context, size *metal.Size) error {
	return rs.createEntity(ctx, rs.sizeTable(), size)
}

// DeleteSize deletes a size.
func (rs *RethinkStore) DeleteSize(ctx context.Context, size *metal.Size) error {
	return rs.deleteEntity(ctx, rs.sizeTable(), size)
}

// UpdateSize updates a size.
func (rs *RethinkStore) UpdateSize(ctx context.Context, oldSize *metal.Size, newSize *metal.Size) error {
	return rs.updateEntity(ctx, rs.sizeTable(), newSize, oldSize)
}

// FromHardware tries to find a size which matches the given hardware specs.
func (rs *RethinkStore) FromHardware(ctx context.Context, hw metal.MachineHardware) (*metal.Size, []*metal.SizeMatchingLog, error) {
	sz, err := rs.ListSizes(ctx)
	if err != nil {
		return nil, nil, err
	}
	if len(sz) < 1 {
		// this should not happen, so we do not return a notfound
		return nil, nil, errors.New("no sizes found in database")
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
