package hashmapstore

import (
	"fmt"
	"time"

	"git.f-i-ts.de/cloud-native/maas/metal-api/pkg/metal"
)

func (h HashmapStore) addDummySizes() {
	for _, size := range DummySizes {
		h.sizes[size.ID] = size
	}
}

func (h HashmapStore) FindSize(id string) (*metal.Size, error) {
	if size, ok := h.sizes[id]; ok {
		return size, nil
	}
	return nil, fmt.Errorf("size with id %q not found", id)
}

func (h HashmapStore) SearchSize() {

}

func (h HashmapStore) ListSizes() []*metal.Size {
	res := make([]*metal.Size, 0)
	for _, size := range h.sizes {
		res = append(res, size)
	}
	return res
}

func (h HashmapStore) CreateSize(size *metal.Size) error {
	// well, check if this id already exist ... but
	// we do not have a database, so this is ok here :-)
	h.sizes[size.ID] = size
	return nil
}

func (h HashmapStore) DeleteSize(id string) (*metal.Size, error) {
	size, ok := h.sizes[id]
	if ok {
		delete(h.sizes, id)
	} else {
		return nil, fmt.Errorf("size with id %q not found", id)
	}
	return size, nil
}

func (h HashmapStore) DeleteSizes() {
	for _, size := range h.sizes {
		delete(h.sizes, size.ID)
	}
}

func (h HashmapStore) UpdateSize(oldSize *metal.Size, newSize *metal.Size) error {
	if !newSize.Changed.Equal(oldSize.Changed) {
		return fmt.Errorf("size with id %q was changed in the meantime", newSize.ID)
	}

	newSize.Created = oldSize.Created
	newSize.Changed = time.Now()

	h.sizes[newSize.ID] = newSize
	return nil
}
