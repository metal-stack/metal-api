package hashmapstore

import (
	"git.f-i-ts.de/cloud-native/maas/metal-api/cmd/metal-api/internal/datastore"
	"git.f-i-ts.de/cloud-native/maas/metal-api/pkg/metal"
	"github.com/inconshreveable/log15"
)

type HashmapStore struct {
	sizes      map[string]*metal.Size
	images     map[string]*metal.Image
	facilities map[string]*metal.Facility
	devices    devicePool
}

func New() *HashmapStore {
	return &HashmapStore{
		sizes:      make(map[string]*metal.Size),
		images:     make(map[string]*metal.Image),
		facilities: make(map[string]*metal.Facility),
		devices: devicePool{
			all:       make(map[string]*metal.Device),
			free:      make(map[string]*metal.Device),
			allocated: make(map[string]*metal.Device),
			waitfor:   make(map[string]datastore.Allocation),
		},
	}
}

func (h HashmapStore) Connect() {
	log15.Info("HashmapStore connected")
}

func (h HashmapStore) Close() error {
	log15.Info("HashmapStore disconnected")
	return nil
}

func (h HashmapStore) AddMockData() {
	h.addDummySizes()
	h.addDummyImages()
	h.addDummyFacilities()
	h.addDummyDevices()
}
