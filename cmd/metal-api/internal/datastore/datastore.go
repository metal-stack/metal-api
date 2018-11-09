package datastore

import (
	"fmt"

	"git.f-i-ts.de/cloud-native/maas/metal-api/metal"
)

var (
	ErrNoDeviceAvailable = fmt.Errorf("no device available")
)

type Datastore interface {
	DeviceStore
	SizeStore
	ImageStore
	SiteStore
	Connect()
	Close() error
}

type Allocation chan metal.Device
type Allocator func(Allocation) error
type CidrAllocator func(uuid, tenant, project, name, description, os string) (string, error)

type DeviceStore interface {
	FindDevice(id string) (*metal.Device, error)
	SearchDevice(projectid string, mac string, free *bool) ([]metal.Device, error)
	ListDevices() ([]metal.Device, error)
	CreateDevice(device *metal.Device) error
	DeleteDevice(id string) (*metal.Device, error)
	UpdateDevice(oldDevice *metal.Device, newDevice *metal.Device) error
	AllocateDevice(name string,
		description string,
		hostname string,
		projectid string,
		site *metal.Site,
		size *metal.Size,
		img *metal.Image,
		sshPubKeys []string,
		tenant string,
		cidrAllocator CidrAllocator,
	) (*metal.Device, error)
	FreeDevice(id string) (*metal.Device, error)
	RegisterDevice(id string, site metal.Site, size metal.Size, hardware metal.DeviceHardware, ipmi metal.IPMI) (*metal.Device, error)
	Wait(id string, alloc Allocator) error
}

type SizeStore interface {
	FindSize(id string) (*metal.Size, error)
	ListSizes() ([]metal.Size, error)
	CreateSize(size *metal.Size) error
	DeleteSize(id string) (*metal.Size, error)
	UpdateSize(oldSize *metal.Size, newSize *metal.Size) error
	FromHardware(hw metal.DeviceHardware) (*metal.Size, error)
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

type SiteStore interface {
	FindSite(id string) (*metal.Site, error)
	ListSites() ([]metal.Site, error)
	CreateSite(size *metal.Site) error
	DeleteSite(id string) (*metal.Site, error)
	UpdateSite(oldSite *metal.Site, newSite *metal.Site) error
}
