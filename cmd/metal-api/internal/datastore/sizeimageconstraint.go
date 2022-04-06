package datastore

import (
	"context"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
)

// FindSizeImageConstraint return a SizeImageConstraint for a given size.
func (rs *RethinkStore) FindSizeImageConstraint(ctx context.Context, sizeID string) (*metal.SizeImageConstraint, error) {
	var ic metal.SizeImageConstraint
	err := rs.findEntityByID(ctx, rs.sizeImageConstraintTable(), &ic, sizeID)
	if err != nil {
		return nil, err
	}
	return &ic, nil
}

// ListSizeImageConstraints returns all SizeImageConstraints.
func (rs *RethinkStore) ListSizeImageConstraints(ctx context.Context) (metal.SizeImageConstraints, error) {
	fls := make(metal.SizeImageConstraints, 0)
	err := rs.listEntities(ctx, rs.sizeImageConstraintTable(), &fls)
	return fls, err
}

// CreateSizeImageConstraint creates a new SizeImageConstraint.
func (rs *RethinkStore) CreateSizeImageConstraint(ctx context.Context, ic *metal.SizeImageConstraint) error {
	return rs.createEntity(ctx, rs.sizeImageConstraintTable(), ic)
}

// DeleteSizeImageConstraint deletes a SizeImageConstraint.
func (rs *RethinkStore) DeleteSizeImageConstraint(ctx context.Context, ic *metal.SizeImageConstraint) error {
	return rs.deleteEntity(ctx, rs.sizeImageConstraintTable(), ic)
}

// UpdateSizeImageConstraint updates a SizeImageConstraint.
func (rs *RethinkStore) UpdateSizeImageConstraint(ctx context.Context, oldSizeImageConstraint *metal.SizeImageConstraint, newSizeImageConstraint *metal.SizeImageConstraint) error {
	return rs.updateEntity(ctx, rs.sizeImageConstraintTable(), newSizeImageConstraint, oldSizeImageConstraint)
}
