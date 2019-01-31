package datastore

import (
	"fmt"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
)

// Some predefined error values.
var (
	ErrNoDeviceAvailable = fmt.Errorf("no device available")
)

// An Allocation is a queue of allocated devices. You can read the devices
// to get the next allocated one.
type Allocation <-chan metal.Device

// An Allocator is a callback for some piece of code if this wants to read
// allocated devices.
type Allocator func(Allocation) error

// A CidrAllocator must return a new CIDR if the allocate method is invoked.
// On the other hand it should release the cidr which is connected to the
// device given with 'uuid' when the Release function is called.
type CidrAllocator interface {
	Allocate(uuid string, tenant string, vrf uint, project, name, description, os string) (string, error)
	Release(uuid string) error
}
