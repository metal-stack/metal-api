package datastore

import (
	"fmt"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
	r "gopkg.in/rethinkdb/rethinkdb-go.v5"
)

// FindImage returns an image or a given id.
func (rs *RethinkStore) FindImage(id string) (*metal.Image, error) {
	res, err := rs.table("image").Get(id).Run(rs.session)
	if err != nil {
		return nil, fmt.Errorf("cannot get image from database: %v", err)
	}
	defer res.Close()
	if res.IsNil() {
		return nil, metal.NotFound("no image %q found", id)
	}
	var img metal.Image
	err = res.One(&img)
	if err != nil {
		return nil, fmt.Errorf("cannot fetch results: %v", err)
	}
	return &img, nil
}

// ListImages returns all images.
func (rs *RethinkStore) ListImages() ([]metal.Image, error) {
	res, err := rs.table("image").Run(rs.session)
	if err != nil {
		return nil, fmt.Errorf("cannot search images from database: %v", err)
	}
	defer res.Close()
	imgs := make([]metal.Image, 0)
	err = res.All(&imgs)
	if err != nil {
		return nil, fmt.Errorf("cannot fetch results: %v", err)
	}
	return imgs, nil
}

// CreateImage creates a new image.
func (rs *RethinkStore) CreateImage(i *metal.Image) (*metal.Image, error) {
	res, err := rs.table("image").Insert(i).RunWrite(rs.session)
	if err != nil {
		return nil, fmt.Errorf("cannot create image in database: %v", err)
	}
	if i.ID == "" {
		i.ID = res.GeneratedKeys[0]
	}
	return i, nil
}

// DeleteImage deletes an image.
func (rs *RethinkStore) DeleteImage(id string) (*metal.Image, error) {
	img, err := rs.FindImage(id)
	if err != nil {
		return nil, err
	}
	_, err = rs.table("image").Get(id).Delete().RunWrite(rs.session)
	if err != nil {
		return nil, fmt.Errorf("cannot delete image from database: %v", err)
	}
	return img, nil
}

// UpdateImage updates an image.
func (rs *RethinkStore) UpdateImage(oldImage *metal.Image, newImage *metal.Image) error {
	_, err := rs.table("image").Get(oldImage.ID).Replace(func(row r.Term) r.Term {
		return r.Branch(row.Field("changed").Eq(r.Expr(oldImage.Changed)), newImage, r.Error("the image was changed from another, please retry"))
	}).RunWrite(rs.session)
	if err != nil {
		return fmt.Errorf("cannot update image: %v", err)
	}
	return nil
}
