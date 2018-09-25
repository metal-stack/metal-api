package hashmapstore

import (
	"fmt"
	"time"

	"git.f-i-ts.de/cloud-native/maas/maas-service/pkg/maas"
)

func (h HashmapStore) addDummyImages() {
	for _, image := range maas.DummyImages {
		h.images[image.ID] = image
	}
}

func (h HashmapStore) FindImage(id string) (*maas.Image, error) {
	if image, ok := h.images[id]; ok {
		return image, nil
	}
	return nil, fmt.Errorf("image with id %q not found", id)
}

func (h HashmapStore) SearchImage() {

}

func (h HashmapStore) ListImages() []*maas.Image {
	res := make([]*maas.Image, 0)
	for _, image := range h.images {
		res = append(res, image)
	}
	return res
}

func (h HashmapStore) CreateImage(image *maas.Image) error {
	// well, check if this id already exist ... but
	// we do not have a database, so this is ok here :-)
	h.images[image.ID] = image
	return nil
}

func (h HashmapStore) DeleteImage(id string) (*maas.Image, error) {
	image, ok := h.images[id]
	if ok {
		delete(h.images, id)
	} else {
		return nil, fmt.Errorf("image with id %q not found", id)
	}
	return image, nil
}

func (h HashmapStore) DeleteImages() {
	for _, image := range h.images {
		delete(h.images, image.ID)
	}
}

func (h HashmapStore) UpdateImage(oldImage *maas.Image, newImage *maas.Image) error {
	if !newImage.Changed.Equal(oldImage.Changed) {
		return fmt.Errorf("image with id %q was changed in the meantime", newImage.ID)
	}

	newImage.Created = oldImage.Created
	newImage.Changed = time.Now()

	h.images[newImage.ID] = newImage
	return nil
}
