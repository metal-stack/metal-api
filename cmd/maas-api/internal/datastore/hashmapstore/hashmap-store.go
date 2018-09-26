package hashmapstore

import (
	"git.f-i-ts.de/cloud-native/maas/maas-service/pkg/maas"
	"github.com/inconshreveable/log15"
)

type HashmapStore struct {
	sizes      map[string]*maas.Size
	images     map[string]*maas.Image
	facilities map[string]*maas.Facility
	devices    devicePool
}

func NewHashmapStore() *HashmapStore {
	return &HashmapStore{
		sizes:      make(map[string]*maas.Size),
		images:     make(map[string]*maas.Image),
		facilities: make(map[string]*maas.Facility),
		devices: devicePool{
			all:       make(map[string]*maas.Device),
			free:      make(map[string]*maas.Device),
			allocated: make(map[string]*maas.Device),
		},
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
	h.addDummyFacilities()
	h.addDummyDevices()
}
