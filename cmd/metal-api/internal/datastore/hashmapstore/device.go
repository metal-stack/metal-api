package hashmapstore

import (
	"fmt"
	"time"

	"git.f-i-ts.de/cloud-native/maas/metal-api/cmd/metal-api/internal/datastore"
	"git.f-i-ts.de/cloud-native/maas/metal-api/pkg/metal"
	"github.com/inconshreveable/log15"
)

type devicePool struct {
	all       map[string]*metal.Device
	free      map[string]*metal.Device
	allocated map[string]*metal.Device
	waitfor   map[string]datastore.Allocation
}

func (h HashmapStore) addDummyDevices() {
	for _, device := range DummyDevices {
		h.devices.all[device.ID] = device
		if device.Name == "" {
			h.devices.free[device.ID] = device
		} else {
			h.devices.allocated[device.ID] = device
		}
	}
}

func (h HashmapStore) Wait(id string, alloc datastore.Allocator) error {
	a := make(datastore.Allocation)
	h.devices.waitfor[id] = a
	defer delete(h.devices.waitfor, id)
	alloc(a)
	return nil
}

func (h HashmapStore) FindDevice(id string) (*metal.Device, error) {
	if device, ok := h.devices.all[id]; ok {
		return device, nil
	}
	return nil, fmt.Errorf("device with id %q not found", id)
}

func (h HashmapStore) SearchDevice(projectid string, mac string, free *bool) ([]metal.Device, error) {
	devicePool := h.devices.all
	if free != nil {
		if *free {
			devicePool = h.devices.free
		} else {
			devicePool = h.devices.allocated
		}
	}

	result := make([]metal.Device, 0)
	for _, d := range devicePool {
		if projectid != "" && d.Project != projectid {
			continue
		}
		if mac != "" && !d.HasMAC(mac) {
			continue
		}
		result = append(result, *d)
	}
	return result, nil
}

func (h HashmapStore) ListDevices() ([]metal.Device, error) {
	res := make([]metal.Device, 0)
	for _, dev := range h.devices.all {
		res = append(res, *dev)
	}
	return res, nil
}

func (h HashmapStore) CreateDevice(device *metal.Device) error {
	// well, check if this id already exist ... but
	// we do not have a database, so this is ok here :-)
	device.Created = time.Now()
	device.Changed = device.Created
	h.devices.all[device.ID] = device
	h.devices.free[device.ID] = device
	return nil
}

func (h HashmapStore) DeleteDevice(id string) (*metal.Device, error) {
	device, ok := h.devices.all[id]
	if ok {
		delete(h.devices.all, id)
	} else {
		return nil, fmt.Errorf("device with id %q not found", id)
	}
	_, ok = h.devices.free[id]
	if ok {
		delete(h.devices.free, id)
	}
	_, ok = h.devices.allocated[id]
	if ok {
		delete(h.devices.allocated, id)
	}
	return device, nil
}

func (h HashmapStore) UpdateDevice(oldDevice *metal.Device, newDevice *metal.Device) error {
	if !newDevice.Changed.Equal(oldDevice.Changed) {
		return fmt.Errorf("device with id %q was changed in the meantime", newDevice.ID)
	}

	newDevice.Created = oldDevice.Created
	newDevice.Changed = time.Now()

	h.devices.all[newDevice.ID] = newDevice
	return nil
}

func (h HashmapStore) AllocateDevice(name string, description string, projectid string, facilityid string, sizeid string, imageid string) (*metal.Device, error) {
	facility, err := h.FindFacility(facilityid)
	if err != nil {
		return nil, fmt.Errorf("facility with id %q not found", facilityid)
	}

	image, err := h.FindImage(imageid)
	if err != nil {
		return nil, fmt.Errorf("image with id %q not found", imageid)
	}

	size, err := h.FindSize(sizeid)
	if err != nil {
		return nil, fmt.Errorf("size with id %q not found", sizeid)
	}

	var device *metal.Device
	for _, freeDevice := range h.devices.free {
		if _, ok := h.devices.waitfor[freeDevice.ID]; !ok {
			log15.Error("device not waiting", "free-id", freeDevice.ID)
			continue
		}
		if freeDevice.Size.ID == size.ID && freeDevice.Facility.ID == facility.ID {
			device = freeDevice
			break
		}
	}
	if device == nil {
		return nil, fmt.Errorf("no free device available for allocation in facility")
	}

	alloc := h.devices.waitfor[device.ID]

	device.Name = name
	device.Project = projectid
	device.Description = description
	device.Image = image
	device.Changed = time.Now()
	// we must set the IP, the network config, ...

	delete(h.devices.free, device.ID)
	alloc <- *device

	h.devices.allocated[device.ID] = device

	return device, nil
}

// func (h HashmapStore) FreeDevice(id string) error {
// 	device, ok := h.devices.all[id]
// 	if !ok {
// 		return fmt.Errorf("device with id %q not found", id)
// 	}

// 	// TODO: Actually the device needs to be deleted completely and then rebooted

// 	device.Name = ""
// 	device.Project = ""
// 	device.Description = ""
// 	device.Facility = metal.Facility{}
// 	device.Image = nil
// 	device.Size = metal.Size{}
// 	device.Changed = time.Now()

// 	delete(h.devices.allocated, id)
// 	h.devices.free[id] = device

// 	return nil
// }

// func (h HashmapStore) RegisterDevice(id string) (*metal.Device, error) {
// 	device, err := h.FindDevice(id)
// 	if err != nil {
// 		device = &metal.Device{
// 			ID:           id,
// 			MACAddresses: macs,
// 			Facility:     *facility,
// 			Size:         *size,
// 		}
// 		err = h.CreateDevice(device)
// 		if err != nil {
// 			return nil, err
// 		}
// 	}

// 	device.MACAddresses = macs
// 	device.Facility = *facility
// 	device.Size = *size

// 	return device, nil
// }
