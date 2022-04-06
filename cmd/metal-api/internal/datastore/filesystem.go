package datastore

import (
	"context"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
)

// FindFilesystemLayout return a filesystemlayout for a given id.
func (rs *RethinkStore) FindFilesystemLayout(ctx context.Context, id string) (*metal.FilesystemLayout, error) {
	var fl metal.FilesystemLayout
	err := rs.findEntityByID(ctx, rs.filesystemLayoutTable(), &fl, id)
	if err != nil {
		return nil, err
	}
	return &fl, nil
}

// ListFilesystemLayouts returns all filesystemlayouts.
func (rs *RethinkStore) ListFilesystemLayouts(ctx context.Context) (metal.FilesystemLayouts, error) {
	fls := make(metal.FilesystemLayouts, 0)
	err := rs.listEntities(ctx, rs.filesystemLayoutTable(), &fls)
	return fls, err
}

// CreateFilesystemLayout creates a new filesystemlayout.
func (rs *RethinkStore) CreateFilesystemLayout(ctx context.Context, fl *metal.FilesystemLayout) error {
	return rs.createEntity(ctx, rs.filesystemLayoutTable(), fl)
}

// DeleteFilesystemLayout deletes a filesystemlayout.
func (rs *RethinkStore) DeleteFilesystemLayout(ctx context.Context, fl *metal.FilesystemLayout) error {
	return rs.deleteEntity(ctx, rs.filesystemLayoutTable(), fl)
}

// UpdateFilesystemLayout updates a filesystemlayout.
func (rs *RethinkStore) UpdateFilesystemLayout(ctx context.Context, oldFilesystemLayout *metal.FilesystemLayout, newFilesystemLayout *metal.FilesystemLayout) error {
	return rs.updateEntity(ctx, rs.filesystemLayoutTable(), newFilesystemLayout, oldFilesystemLayout)
}
