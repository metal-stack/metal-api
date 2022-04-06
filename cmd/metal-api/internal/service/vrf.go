package service

import (
	"context"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
)

// acquireRandomVRF will grab a unique but random vrf out of the vrfintegerpool
func acquireRandomVRF(ctx context.Context, ds *datastore.RethinkStore) (*uint, error) {
	vrf, err := ds.GetVRFPool().AcquireRandomUniqueInteger(ctx)
	return &vrf, err
}

// acquireVRF will the given vrf out of the vrfintegerpool if not available a error is thrown
func acquireVRF(ctx context.Context, ds *datastore.RethinkStore, vrf uint) error {
	_, err := ds.GetVRFPool().AcquireUniqueInteger(ctx, vrf)
	return err
}

// releaseVRF will return the given vrf to the vrfintegerpool for reuse
func releaseVRF(ds *datastore.RethinkStore, vrf uint) error {
	return ds.GetVRFPool().ReleaseUniqueInteger(vrf)
}
