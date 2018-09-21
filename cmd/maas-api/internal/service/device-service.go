package service

import (
	"git.f-i-ts.de/ize0h88/maas-service/pkg/maas"
)

type DevicePool struct {
	Free      map[string]maas.Device
	Allocated map[string]maas.Device
}
