package service

import (
	"fmt"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
)

// acquireRandomVRF will grab a unique but random vrf out of the vrfintegerpool
func acquireRandomVRF(ds *datastore.RethinkStore) (*uint, error) {
	vrfPool, err := ds.GetIntegerPool(datastore.VRFIntegerPool)
	if err != nil {
		return nil, fmt.Errorf("could not acquire random vrf: %v", err)
	}
	vrf, err := vrfPool.AcquireRandomUniqueInteger()
	return &vrf, err
}

// acquireVRF will the given vrf out of the vrfintegerpool if not available a error is thrown
func acquireVRF(ds *datastore.RethinkStore, vrf uint) error {
	vrfPool, err := ds.GetIntegerPool(datastore.VRFIntegerPool)
	if err != nil {
		return fmt.Errorf("could not acquire vrf:%d %v", vrf, err)
	}
	_, err = vrfPool.AcquireUniqueInteger(vrf)
	return err
}

// releaseVRF will return the given vrf to the vrfintegerpool for reuse
func releaseVRF(ds *datastore.RethinkStore, vrf uint) error {
	vrfPool, err := ds.GetIntegerPool(datastore.VRFIntegerPool)
	if err != nil {
		return fmt.Errorf("could not release vrf: %v", err)
	}
	return vrfPool.ReleaseUniqueInteger(vrf)
}
