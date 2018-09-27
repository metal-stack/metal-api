package datastore

import "git.f-i-ts.de/cloud-native/maas/metal-api/pkg/metal"

type Datastore interface {
	DeviceStore
	SizeStore
	ImageStore
	FacilityStore
	Connect()
	Close()
	AddMockData()
}

type Allocation chan metal.Device
type Allocator func(Allocation) error

type DeviceStore interface {
	FindDevice(id string) (*metal.Device, error)
	SearchDevice(projectid string, mac string, pool string) []*metal.Device
	ListDevices() []*metal.Device
	CreateDevice(device *metal.Device) error
	DeleteDevice(id string) (*metal.Device, error)
	UpdateDevice(oldDevice *metal.Device, newDevice *metal.Device) error
	AllocateDevice(name string, description string, projectid string, facilityid string, sizeid string, imageid string) error
	FreeDevice(id string) error
	RegisterDevice(id string, macs []string, facilityid string, sizeid string) (*metal.Device, error)
	Wait(id string, alloc Allocator)
}

type SizeStore interface {
	FindSize(id string) (*metal.Size, error)
	SearchSize()
	ListSizes() []*metal.Size
	CreateSize(size *metal.Size) error
	DeleteSize(id string) (*metal.Size, error)
	DeleteSizes()
	UpdateSize(oldSize *metal.Size, newSize *metal.Size) error
}

type ImageStore interface {
	FindImage(id string) (*metal.Image, error)
	SearchImage()
	ListImages() []*metal.Image
	CreateImage(size *metal.Image) error
	DeleteImage(id string) (*metal.Image, error)
	DeleteImages()
	UpdateImage(oldImage *metal.Image, newImage *metal.Image) error
}

type FacilityStore interface {
	FindFacility(id string) (*metal.Facility, error)
	SearchFacility()
	ListFacilities() []*metal.Facility
	CreateFacility(facility *metal.Facility) error
	DeleteFacility(id string) (*metal.Facility, error)
	DeleteFacilities()
	UpdateFacility(oldFacility *metal.Facility, newFacility *metal.Facility) error
}
