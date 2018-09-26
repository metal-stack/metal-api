package datastore

import (
	"git.f-i-ts.de/cloud-native/maas/maas-service/pkg/maas"
)

type Datastore interface {
	DeviceStore
	SizeStore
	ImageStore
	FacilityStore
	Connect()
	Close()
	AddMockData()
}

type DeviceStore interface {
	FindDevice(id string) (*maas.Device, error)
	SearchDevice(projectid string, mac string, pool string) []*maas.Device
	ListDevices() []*maas.Device
	CreateDevice(device *maas.Device) error
	DeleteDevice(id string) (*maas.Device, error)
	UpdateDevice(oldDevice *maas.Device, newDevice *maas.Device) error
	AllocateDevice(name string, description string, projectid string, facilityid string, sizeid string, imageid string) error
	FreeDevice(id string) error
	RegisterDevice(id string, macs []string, facilityid string, sizeid string) (*maas.Device, error)
}

type SizeStore interface {
	FindSize(id string) (*maas.Size, error)
	SearchSize()
	ListSizes() []*maas.Size
	CreateSize(size *maas.Size) error
	DeleteSize(id string) (*maas.Size, error)
	DeleteSizes()
	UpdateSize(oldSize *maas.Size, newSize *maas.Size) error
}

type ImageStore interface {
	FindImage(id string) (*maas.Image, error)
	SearchImage()
	ListImages() []*maas.Image
	CreateImage(size *maas.Image) error
	DeleteImage(id string) (*maas.Image, error)
	DeleteImages()
	UpdateImage(oldImage *maas.Image, newImage *maas.Image) error
}

type FacilityStore interface {
	FindFacility(id string) (*maas.Facility, error)
	SearchFacility()
	ListFacilities() []*maas.Facility
	CreateFacility(facility *maas.Facility) error
	DeleteFacility(id string) (*maas.Facility, error)
	DeleteFacilities()
	UpdateFacility(oldFacility *maas.Facility, newFacility *maas.Facility) error
}
