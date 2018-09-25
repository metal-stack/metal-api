package datastore

import (
	"git.f-i-ts.de/cloud-native/maas/maas-service/pkg/maas"
)

type Datastore interface {
	Connect()
	Close()
	AddMockData()
	// Size
	FindSize(id string) (*maas.Size, error)
	SearchSize()
	ListSizes() []*maas.Size
	CreateSize(size *maas.Size) error
	DeleteSize(id string) (*maas.Size, error)
	DeleteSizes()
	UpdateSize(oldSize *maas.Size, newSize *maas.Size) error
	// Image
	FindImage(id string) (*maas.Image, error)
	SearchImage()
	ListImages() []*maas.Image
	CreateImage(size *maas.Image) error
	DeleteImage(id string) (*maas.Image, error)
	DeleteImages()
	UpdateImage(oldImage *maas.Image, newImage *maas.Image) error
}
