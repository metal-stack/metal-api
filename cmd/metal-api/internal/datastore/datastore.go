package datastore

import (
	"fmt"

	"git.f-i-ts.de/cloud-native/metal/metal-api/metal"
)

// Some predefined error values.
var (
	ErrNoDeviceAvailable = fmt.Errorf("no device available")
	ErrNotFound          = fmt.Errorf("no entity found")
)

type Allocation chan metal.Device
type Allocator func(Allocation) error
type CidrAllocator func(uuid, tenant, project, name, description, os string) (string, error)
