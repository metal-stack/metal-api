package datastore

import "github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"

// FindSizeImageConstraint return a SizeImageConstraint for a given size.
func (rs *RethinkStore) FindSizeImageConstraint(sizeID string) (*metal.SizeImageConstraint, error) {
	var ic metal.SizeImageConstraint
	err := rs.findEntityByID(rs.sizeImageConstraintTable(), &ic, sizeID)
	if err != nil {
		return nil, err
	}
	return &ic, nil
}

// ListSizeImageConstraints returns all SizeImageConstraints.
func (rs *RethinkStore) ListSizeImageConstraints() (metal.SizeImageConstraints, error) {
	fls := make(metal.SizeImageConstraints, 0)
	err := rs.listEntities(rs.sizeImageConstraintTable(), &fls)
	return fls, err
}

// CreateSizeImageConstraint creates a new SizeImageConstraint.
func (rs *RethinkStore) CreateSizeImageConstraint(ic *metal.SizeImageConstraint) error {
	return rs.createEntity(rs.sizeImageConstraintTable(), ic)
}

// DeleteSizeImageConstraint deletes a SizeImageConstraint.
func (rs *RethinkStore) DeleteSizeImageConstraint(ic *metal.SizeImageConstraint) error {
	return rs.deleteEntity(rs.sizeImageConstraintTable(), ic)
}

// UpdateSizeImageConstraint updates a SizeImageConstraint.
func (rs *RethinkStore) UpdateSizeImageConstraint(oldSizeImageConstraint *metal.SizeImageConstraint, newSizeImageConstraint *metal.SizeImageConstraint) error {
	return rs.updateEntity(rs.sizeImageConstraintTable(), newSizeImageConstraint, oldSizeImageConstraint)
}
