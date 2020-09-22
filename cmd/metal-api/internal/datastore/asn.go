package datastore

import (
	"fmt"
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

// AcquireASN fetches a unique integer by using the existing integer pool and adding to ASNBase
func (rs *RethinkStore) AcquireASN() (*uint32, error) {

	i, err := rs.AcquireRandomUniqueInteger()
	if err != nil {
		return nil, err
	}
	asn := ASNBase + uint32(i)
	if asn > ASNMax {
		return nil, fmt.Errorf("unable to calculate asn, got a asn larger than ASNMax: %d > %d", asn, ASNMax)
	}
	return &asn, nil
}

// ReleaseASN will release the asn from the integerpool
func (rs *RethinkStore) ReleaseASN(asn uint32) error {
	if asn < ASNBase {
		return fmt.Errorf("asn might not be smaller than:%d", ASNBase)
	}
	i := uint(asn - ASNBase)

	return rs.ReleaseUniqueInteger(i)
}
