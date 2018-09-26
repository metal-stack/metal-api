package hashmapstore

import (
	"fmt"
	"time"

	"git.f-i-ts.de/cloud-native/maas/metal-api/pkg/metal"
)

func (h HashmapStore) addDummyFacilities() {
	for _, facility := range metal.DummyFacilities {
		h.facilities[facility.ID] = facility
	}
}

func (h HashmapStore) FindFacility(id string) (*metal.Facility, error) {
	if facility, ok := h.facilities[id]; ok {
		return facility, nil
	}
	return nil, fmt.Errorf("facility with id %q not found", id)
}

func (h HashmapStore) SearchFacility() {

}

func (h HashmapStore) ListFacilities() []*metal.Facility {
	res := make([]*metal.Facility, 0)
	for _, facility := range h.facilities {
		res = append(res, facility)
	}
	return res
}

func (h HashmapStore) CreateFacility(facility *metal.Facility) error {
	// well, check if this id already exist ... but
	// we do not have a database, so this is ok here :-)
	h.facilities[facility.ID] = facility
	return nil
}

func (h HashmapStore) DeleteFacility(id string) (*metal.Facility, error) {
	facility, ok := h.facilities[id]
	if ok {
		delete(h.facilities, id)
	} else {
		return nil, fmt.Errorf("facility with id %q not found", id)
	}
	return facility, nil
}

func (h HashmapStore) DeleteFacilities() {
	for _, facility := range h.facilities {
		delete(h.facilities, facility.ID)
	}
}

func (h HashmapStore) UpdateFacility(oldFacility *metal.Facility, newFacility *metal.Facility) error {
	if !newFacility.Changed.Equal(oldFacility.Changed) {
		return fmt.Errorf("facility with id %q was changed in the meantime", newFacility.ID)
	}

	newFacility.Created = oldFacility.Created
	newFacility.Changed = time.Now()

	h.facilities[newFacility.ID] = newFacility
	return nil
}
