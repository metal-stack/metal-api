package datastore

import "github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"

// FindFilesystemLayout return a size for a given id.
func (rs *RethinkStore) FindFilesystemLayout(id string) (*metal.FilesystemLayout, error) {
	var fl metal.FilesystemLayout
	err := rs.findEntityByID(rs.filesystemLayoutTable(), &fl, id)
	if err != nil {
		return nil, err
	}
	return &fl, nil
}

// ListFilesystemLayouts returns all sizes.
func (rs *RethinkStore) ListFilesystemLayouts() (metal.FilesystemLayouts, error) {
	fls := make(metal.FilesystemLayouts, 0)
	err := rs.listEntities(rs.filesystemLayoutTable(), &fls)
	return fls, err
}

// CreateFilesystemLayout creates a new size.
func (rs *RethinkStore) CreateFilesystemLayout(fl *metal.FilesystemLayout) error {
	return rs.createEntity(rs.filesystemLayoutTable(), fl)
}

// DeleteFilesystemLayout deletes a size.
func (rs *RethinkStore) DeleteFilesystemLayout(fl *metal.FilesystemLayout) error {
	return rs.deleteEntity(rs.filesystemLayoutTable(), fl)
}

// UpdateFilesystemLayout updates a size.
func (rs *RethinkStore) UpdateFilesystemLayout(oldFilesystemLayout *metal.FilesystemLayout, newFilesystemLayout *metal.FilesystemLayout) error {
	return rs.updateEntity(rs.filesystemLayoutTable(), newFilesystemLayout, oldFilesystemLayout)
}
