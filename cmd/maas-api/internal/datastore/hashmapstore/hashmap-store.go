package hashmapstore

import (
	"git.f-i-ts.de/cloud-native/maas/maas-service/pkg/maas"
	"github.com/inconshreveable/log15"
)

type HashmapStore struct {
	sizes  map[string]*maas.Size
	images map[string]*maas.Image
}

func NewHashmapStore() *HashmapStore {
	return &HashmapStore{
		sizes:  make(map[string]*maas.Size),
		images: make(map[string]*maas.Image),
	}
}

func (h HashmapStore) Connect() {
	log15.Info("HashmapStore connected")
}

func (h HashmapStore) Close() {
	log15.Info("HashmapStore disconnected")
}

func (h HashmapStore) AddMockData() {
	h.addDummySizes()
	h.addDummyImages()
}
