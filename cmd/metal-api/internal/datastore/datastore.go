package datastore

import (
	"fmt"

	"git.f-i-ts.de/cloud-native/maas/metal-api/pkg/metal"
)

var (
	ErrNoDeviceAvailable = fmt.Errorf("no device available")
)

type Datastore interface {
	DeviceStore
	SizeStore
	ImageStore
	FacilityStore
	Connect()
	Close() error
}

type Allocation chan metal.Device
type Allocator func(Allocation) error

type DeviceStore interface {
	FindDevice(id string) (*metal.Device, error)
	SearchDevice(projectid string, mac string, free *bool) ([]metal.Device, error)
	ListDevices() ([]metal.Device, error)
	CreateDevice(device *metal.Device) error
	DeleteDevice(id string) (*metal.Device, error)
	UpdateDevice(oldDevice *metal.Device, newDevice *metal.Device) error
	AllocateDevice(name string, description string, hostname string, projectid string, facilityid string, sizeid string, imageid string, sshPubKey string) (*metal.Device, error)
	FreeDevice(id string) (*metal.Device, error)
	RegisterDevice(id string, facilityid string, hardware metal.DeviceHardware) (*metal.Device, error)
	Wait(id string, alloc Allocator) error
}

type SizeStore interface {
	FindSize(id string) (*metal.Size, error)
	SearchSize() error
	ListSizes() ([]metal.Size, error)
	CreateSize(size *metal.Size) error
	DeleteSize(id string) (*metal.Size, error)
	DeleteSizes() error
	UpdateSize(oldSize *metal.Size, newSize *metal.Size) error
}

type ImageStore interface {
	FindImage(id string) (*metal.Image, error)
	SearchImage() error
	ListImages() ([]metal.Image, error)
	CreateImage(size *metal.Image) (*metal.Image, error)
	DeleteImage(id string) (*metal.Image, error)
	DeleteImages() error
	UpdateImage(oldImage *metal.Image, newImage *metal.Image) error
}

type FacilityStore interface {
	FindFacility(id string) (*metal.Facility, error)
	SearchFacility() error
	ListFacilities() ([]metal.Facility, error)
	CreateFacility(facility *metal.Facility) error
	DeleteFacility(id string) (*metal.Facility, error)
	DeleteFacilities() error
	UpdateFacility(oldFacility *metal.Facility, newFacility *metal.Facility) error
}
