package hashmapstore

import (
	"fmt"
	"time"

	"git.f-i-ts.de/cloud-native/maas/metal-api/pkg/metal"
)

func (h HashmapStore) addDummyImages() {
	for _, image := range DummyImages {
		h.images[image.ID] = image
	}
}

func (h HashmapStore) FindImage(id string) (*metal.Image, error) {
	if image, ok := h.images[id]; ok {
		return image, nil
	}
	return nil, fmt.Errorf("image with id %q not found", id)
}

func (h HashmapStore) SearchImage() error {
	return nil
}

func (h HashmapStore) ListImages() ([]metal.Image, error) {
	res := make([]metal.Image, 0)
	for _, image := range h.images {
		res = append(res, *image)
	}
	return res, nil
}

func (h HashmapStore) CreateImage(image *metal.Image) (*metal.Image, error) {
	// well, check if this id already exist ... but
	// we do not have a database, so this is ok here :-)
	h.images[image.ID] = image
	return image, nil
}

func (h HashmapStore) DeleteImage(id string) (*metal.Image, error) {
	image, ok := h.images[id]
	if ok {
		delete(h.images, id)
	} else {
		return nil, fmt.Errorf("image with id %q not found", id)
	}
	return image, nil
}

func (h HashmapStore) DeleteImages() error {
	for _, image := range h.images {
		delete(h.images, image.ID)
	}
	return nil
}

func (h HashmapStore) UpdateImage(oldImage *metal.Image, newImage *metal.Image) error {
	if !newImage.Changed.Equal(oldImage.Changed) {
		return fmt.Errorf("image with id %q was changed in the meantime", newImage.ID)
	}

	newImage.Created = oldImage.Created
	newImage.Changed = time.Now()

	h.images[newImage.ID] = newImage
	return nil
}
