package datastore

import (
	"git.f-i-ts.de/cloud-native/maas/maas-service/pkg/maas"
)

type Datastore interface {
	Connect()
	Close()
	AddMockData()
	FindSize(id string) (*maas.Size, error)
	SearchSize()
	ListSizes() []*maas.Size
	CreateSize(size *maas.Size) error
	DeleteSize(id string) (*maas.Size, error)
	DeleteSizes()
	UpdateSize(oldSize *maas.Size, newSize *maas.Size) error
}
