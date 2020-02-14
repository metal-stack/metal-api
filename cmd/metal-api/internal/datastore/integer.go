package datastore

import (
	"fmt"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"

	r "gopkg.in/rethinkdb/rethinkdb-go.v5"
)

var (
	// IntegerPoolRangeMin the minimum integer to get from the pool
	IntegerPoolRangeMin = uint(1)
	// IntegerPoolRangeMax the maximum integer to get from the pool
	IntegerPoolRangeMax = uint(131072)
)

type integer struct {
	ID uint `rethinkdb:"id"`
}

// Integerinfo contains information on the integer pool.
type Integerinfo struct {
	IsInitialized bool `rethinkdb:"isInitialized"`
}

// initIntegerPool initializes a pool to acquire unique integers from.
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
func (rs *RethinkStore) initIntegerPool() error {

	var result Integerinfo
	err := rs.findEntityByID(rs.integerInfoTable(), &result, "integerpool")
	if err != nil {
		if !metal.IsNotFound(err) {
			return err
		}
	}

	if result.IsInitialized {
		return nil
	}

	rs.SugaredLogger.Infow("Initializing integer pool", "RangeMin", IntegerPoolRangeMin, "RangeMax", IntegerPoolRangeMax)
	intRange := makeRange(IntegerPoolRangeMin, IntegerPoolRangeMax)
	_, err = rs.integerTable().Insert(intRange).RunWrite(rs.session, r.RunOpts{ArrayLimit: IntegerPoolRangeMax})
	if err != nil {
		return err
	}
	_, err = rs.integerInfoTable().Insert(map[string]interface{}{"id": "integerpool", "IsInitialized": true}).RunWrite(rs.session)
	return err
}

// AcquireRandomUniqueInteger returns a random unique integer from the pool.
func (rs *RethinkStore) AcquireRandomUniqueInteger() (uint, error) {
	t := rs.integerTable().Limit(1)
	return rs.genericAcquire(&t)
}

// AcquireUniqueInteger returns a unique integer from the pool.
func (rs *RethinkStore) AcquireUniqueInteger(value uint) (uint, error) {
	err := verifyRange(value)
	if err != nil {
		return 0, err
	}
	t := rs.integerTable().Get(value)
	return rs.genericAcquire(&t)
}

// ReleaseUniqueInteger returns a unique integer to the pool.
func (rs *RethinkStore) ReleaseUniqueInteger(id uint) error {
	err := verifyRange(id)
	if err != nil {
		return err
	}

	i := integer{
		ID: id,
	}
	_, err = rs.integerTable().Insert(i, r.InsertOpts{Conflict: "replace"}).RunWrite(rs.session)
	if err != nil {
		return err
	}

	return nil
}

func (rs *RethinkStore) genericAcquire(term *r.Term) (uint, error) {
	res, err := term.Delete(r.DeleteOpts{ReturnChanges: true}).RunWrite(rs.session)
	if err != nil {
		return 0, err
	}

	if len(res.Changes) == 0 {
		res, err := rs.integerTable().Count().Run(rs.session)
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
		} else {
			return 0, metal.Conflict("integer is already acquired by another")
		}
	}

	result := uint(res.Changes[0].OldValue.(map[string]interface{})["id"].(float64))
	return result, nil
}

func makeRange(min, max uint) []integer {
	a := make([]integer, max-min+1)
	for i := range a {
		a[i] = integer{
			ID: min + uint(i),
		}
	}
	return a
}

func verifyRange(value uint) error {
	if value < IntegerPoolRangeMin || value > IntegerPoolRangeMax {
		return fmt.Errorf("value '%d' is outside of the allowed range '%d - %d'", value, IntegerPoolRangeMin, IntegerPoolRangeMax)
	}
	return nil
}
