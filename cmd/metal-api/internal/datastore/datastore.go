package datastore

import (
	"fmt"

	"git.f-i-ts.de/cloud-native/metal/metal-api/metal"
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

type CidrAllocator func(uuid, tenant, project, name, description, os string) (string, error)
