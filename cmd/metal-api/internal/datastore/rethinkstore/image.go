package rethinkstore

import (
	"fmt"

	"git.f-i-ts.de/cloud-native/maas/metal-api/metal"
	r "gopkg.in/rethinkdb/rethinkdb-go.v5"
)

func (rs *RethinkStore) FindImage(id string) (*metal.Image, error) {
	res, err := rs.imageTable().Get(id).Run(rs.session)
	if err != nil {
		return nil, fmt.Errorf("cannot get image from database: %v", err)
	}
	defer res.Close()
	var img metal.Image
	err = res.One(&img)
	if err != nil {
		return nil, fmt.Errorf("cannot fetch results: %v", err)
	}
	return &img, nil
}

func (rs *RethinkStore) SearchImage() error {
	return fmt.Errorf("not implemented yet")
}

func (rs *RethinkStore) ListImages() ([]metal.Image, error) {
	res, err := rs.imageTable().Run(rs.session)
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

func (rs *RethinkStore) CreateImage(i *metal.Image) (*metal.Image, error) {
	res, err := rs.imageTable().Insert(i).RunWrite(rs.session)
	if err != nil {
		return nil, fmt.Errorf("cannot create image in database: %v", err)
	}
	if i.ID == "" {
		i.ID = res.GeneratedKeys[0]
	}
	return i, nil
}

func (rs *RethinkStore) DeleteImage(id string) (*metal.Image, error) {
	img, err := rs.FindImage(id)
	if err != nil {
		return nil, fmt.Errorf("cannot find image with id %q: %v", id, err)
	}
	_, err = rs.imageTable().Get(id).Delete().RunWrite(rs.session)
	if err != nil {
		return nil, fmt.Errorf("cannot delete image from database: %v", err)
	}
	return img, nil
}

func (rs *RethinkStore) DeleteImages() error {
	// we do not support this here! do we really need such a method?
	return nil
}

func (rs *RethinkStore) UpdateImage(oldImage *metal.Image, newImage *metal.Image) error {
	_, err := rs.imageTable().Get(oldImage.ID).Replace(func(row r.Term) r.Term {
		return r.Branch(row.Field("changed").Eq(r.Expr(oldImage.Changed)), newImage, r.Error("the image was changed from another, please retry"))
	}).RunWrite(rs.session)
	if err != nil {
		return fmt.Errorf("cannot update image: %v", err)
	}
	return nil
}
