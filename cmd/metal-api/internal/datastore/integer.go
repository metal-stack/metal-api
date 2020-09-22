package datastore

import (
	"fmt"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"

	r "gopkg.in/rethinkdb/rethinkdb-go.v6"
)

var (
	// IntegerPoolRangeMin the minimum integer to get from the pool
	IntegerPoolRangeMin = uint32(1)
	// IntegerPoolRangeMax the maximum integer to get from the pool
	IntegerPoolRangeMax = uint32(131072)
)

// IntegerPool defines a pool if integers for unique integer acquisition per use case e.g. name
type IntegerPool struct {
	name string
	min  uint32
	max  uint32
	rs   *RethinkStore
}

type integer struct {
	// TODO: howto migrate to new Schema
	Name string `rethinkdb:"name" json:"name"`
	ID   uint32 `rethinkdb:"id" json:"id"`
}

// Integerinfo contains information on the integer pool.
type integerinfo struct {
	// TODO: howto migrate to new Schema
	Name          string `rethinkdb:"name" json:"name"`
	IsInitialized bool   `rethinkdb:"isInitialized" json:"isInitialized"`
}

// GetIntegerPool returns a named integerpool if already created
func (rs *RethinkStore) GetIntegerPool(name string) (*IntegerPool, error) {
	ip, ok := rs.IntegerPools[name]
	if !ok {
		return nil, fmt.Errorf("no integerpool for %s created", name)
	}
	return ip, nil
}

// NewIntegerPool initializes a pool to acquire unique integers from.
// the acquired integers are used from the network service for defining the:
// - vrf name
// - vni
// - vxlan-id
//
// the integer range has a vxlan-id constraint from Cumulus:
// 	net add vxlan vxlan10 vxlan id
//  <1-16777214>  :  An integer from 1 to 16777214
//
// in order not to impact performance too much, we limited the range of integers to 2^17=131072,
// which includes the range that we typically used for vrf names in the past.
//
// the implementation of the pool works as follows:
// - write the given range of integers to the rethinkdb on first start (with the integer as the document id)
// - write a marker that the pool was initialized to another table (integerpoolinfo), such that it will not initialize again
// - to acquire an integer, delete a random document from the pool and return it to the caller
// - to give it back, a caller can insert the integer back into the database
//
// implementing an efficient unique pool of integers is not so easy.
// the current implementation comes with a performance cost during initialization of the database.
// the initialization takes a few seconds but only needs to be run once in a lifetime of database.
// this seems to be a reasonable trade-off as the following positive criteria are guaranteed:
// - acquiring an integer is fast
// - releasing the integer is fast
// - you do not have gaps (because you can give the integers back to the pool)
// - everything can be done atomically, so there are no race conditions
func (rs *RethinkStore) NewIntegerPool(name string, min, max uint32) (*IntegerPool, error) {
	var result []integerinfo
	// Search by name also
	err := rs.findEntityByID(rs.integerInfoTable(), &result, "integerpool")
	if err != nil {
		if !metal.IsNotFound(err) {
			return nil, err
		}
	}

	ip := &IntegerPool{
		name: name,
		min:  min,
		max:  max,
		rs:   rs,
	}
	for _, r := range result {
		if r.Name == name && r.IsInitialized {
			return ip, nil
		}
	}

	rs.SugaredLogger.Infow("Initializing integer pool", "RangeMin", min, "RangeMax", max)
	intRange := makeRange(name, min, max)
	_, err = rs.integerTable().Insert(intRange).RunWrite(rs.session, r.RunOpts{ArrayLimit: max})
	if err != nil {
		return nil, err
	}
	_, err = rs.integerInfoTable().Insert(map[string]interface{}{"name": name, "id": "integerpool", "IsInitialized": true}).RunWrite(rs.session)
	return ip, err
}

// AcquireRandomUniqueInteger returns a random unique integer from the pool.
func (ip *IntegerPool) AcquireRandomUniqueInteger() (uint32, error) {
	i := integer{
		Name: ip.name,
	}
	t := ip.rs.integerTable().Get(i).Limit(1)
	return ip.genericAcquire(&t)
}

// AcquireUniqueInteger returns a unique integer from the pool.
func (ip *IntegerPool) AcquireUniqueInteger(value uint32) (uint32, error) {
	err := ip.verifyRange(value)
	if err != nil {
		return 0, err
	}
	i := integer{
		Name: ip.name,
		ID:   value,
	}
	t := ip.rs.integerTable().Get(i)
	return ip.genericAcquire(&t)
}

// ReleaseUniqueInteger returns a unique integer to the pool.
func (ip *IntegerPool) ReleaseUniqueInteger(id uint32) error {
	err := ip.verifyRange(id)
	if err != nil {
		return err
	}

	i := integer{
		Name: ip.name,
		ID:   id,
	}
	_, err = ip.rs.integerTable().Insert(i, r.InsertOpts{Conflict: "replace"}).RunWrite(ip.rs.session)
	if err != nil {
		return err
	}

	return nil
}

func (ip *IntegerPool) genericAcquire(term *r.Term) (uint32, error) {
	res, err := term.Delete(r.DeleteOpts{ReturnChanges: true}).RunWrite(ip.rs.session)
	if err != nil {
		return 0, err
	}

	if len(res.Changes) == 0 {
		res, err := ip.rs.integerTable().Count().Run(ip.rs.session)
		if err != nil {
			return 0, err
		}
		var count int64
		err = res.One(&count)
		if err != nil {
			return 0, err
		}

		if count <= 0 {
			return 0, metal.Internal(fmt.Errorf("acquisition of a value failed for exhausted pool"), "")
		}
		return 0, metal.Conflict("integer is already acquired by another")
	}

	result := uint32(res.Changes[0].OldValue.(map[string]interface{})["id"].(float64))
	return result, nil
}

func makeRange(name string, min, max uint32) []integer {
	a := make([]integer, max-min+1)
	for i := range a {
		a[i] = integer{
			Name: name,
			ID:   min + uint32(i),
		}
	}
	return a
}

func (ip *IntegerPool) verifyRange(value uint32) error {
	if value < ip.min || value > ip.max {
		return fmt.Errorf("value '%d' is outside of the allowed range '%d - %d'", value, ip.min, ip.max)
	}
	return nil
}
