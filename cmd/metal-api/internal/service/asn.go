package service

import (
	"fmt"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
)

const (
	// ASNMin is the minimum asn defined according to
	// https://en.wikipedia.org/wiki/Autonomous_system_(Internet)
	ASNMin = uint32(4200000000)

	// ASNBase is the offset for all Machine ASN´s
	ASNBase = uint32(4210000000)

	// ASNMax defines the maximum allowed asn
	// https://en.wikipedia.org/wiki/Autonomous_system_(Internet)
	ASNMax = uint32(4294967294)
)

// acquireASN fetches a unique integer by using the existing integer pool and adding to ASNBase
func acquireASN(ds *datastore.RethinkStore) (*uint32, error) {
	i, err := ds.GetASNPool().AcquireRandomUniqueInteger()
	if err != nil {
		return nil, err
	}
	asn := ASNBase + uint32(i) // nolint:gosec
	if asn > ASNMax {
		return nil, fmt.Errorf("unable to calculate asn, got a asn larger than ASNMax: %d > %d", asn, ASNMax)
	}
	return &asn, nil
}

// releaseASN will release the asn from the integerpool
func releaseASN(ds *datastore.RethinkStore, asn uint32) error {
	if asn < ASNBase || asn > ASNMax {
		return fmt.Errorf("asn %d might not be smaller than:%d or larger than %d", asn, ASNBase, ASNMax)
	}
	i := uint(asn - ASNBase)

	return ds.GetASNPool().ReleaseUniqueInteger(i)
}
