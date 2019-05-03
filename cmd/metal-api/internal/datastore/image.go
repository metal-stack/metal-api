package datastore

import (
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
)

// FindImage returns an image or a given id.
func (rs *RethinkStore) FindImage(id string) (*metal.Image, error) {
	var img metal.Image
	err := rs.findEntityByID(rs.imageTable(), &img, id)
	return &img, err
}

// ListImages returns all images.
func (rs *RethinkStore) ListImages() ([]metal.Image, error) {
	imgs := make([]metal.Image, 0)
	err := rs.listEntities(rs.imageTable(), &imgs)
	return imgs, err
}

// CreateImage creates a new image.
func (rs *RethinkStore) CreateImage(i *metal.Image) error {
	return rs.createEntity(rs.imageTable(), i)
}

// DeleteImage deletes an image.
func (rs *RethinkStore) DeleteImage(i *metal.Image) error {
	return rs.deleteEntityByID(rs.imageTable(), i.GetID())
}

// UpdateImage updates an image.
func (rs *RethinkStore) UpdateImage(oldImage *metal.Image, newImage *metal.Image) error {
	return rs.updateEntity(rs.imageTable(), newImage, oldImage)
}
