package datastore

import (
	"fmt"
	"time"

	"git.f-i-ts.de/cloud-native/maas/maas-service/pkg/maas"
	"github.com/inconshreveable/log15"
	"github.com/spf13/viper"
)

type HashmapStore struct {
	sizes map[string]*maas.Size
}

func NewHashmapStore() *HashmapStore {
	return &HashmapStore{
		sizes: make(map[string]*maas.Size),
	}
}

func (h HashmapStore) Connect() {
	if viper.GetBool("with-mock-data") {
		h.AddMockData()
		log15.Info("Initialized mock data")
	}

	log15.Info("HashmapStore connected")
}

func (h HashmapStore) AddMockData() {
	h.addDummySizes()
}

func (h HashmapStore) addDummySizes() {
	for _, size := range maas.DummySizes {
		h.sizes[size.ID] = size
	}
}

func (h HashmapStore) Close() {
	log15.Info("HashmapStore disconnected")
}

func (h HashmapStore) FindSize(id string) (*maas.Size, error) {
	if size, ok := h.sizes[id]; ok {
		return size, nil
	}
	return nil, fmt.Errorf("size with id %q not found", id)
}

func (h HashmapStore) SearchSize() {

}

func (h HashmapStore) ListSizes() []*maas.Size {
	res := make([]*maas.Size, 0)
	for _, size := range h.sizes {
		res = append(res, size)
	}
	return res
}

func (h HashmapStore) CreateSize(size *maas.Size) error {
	// well, check if this id already exist ... but
	// we do not have a database, so this is ok here :-)
	h.sizes[size.ID] = size
	return nil
}

func (h HashmapStore) DeleteSize(id string) (*maas.Size, error) {
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

func (h HashmapStore) UpdateSize(oldSize *maas.Size, newSize *maas.Size) error {
	if !newSize.Changed.Equal(oldSize.Changed) {
		return fmt.Errorf("size with id %q was changed in the meantime", newSize.ID)
	}

	newSize.Created = oldSize.Created
	newSize.Changed = time.Now()

	h.sizes[newSize.ID] = newSize
	return nil
}
