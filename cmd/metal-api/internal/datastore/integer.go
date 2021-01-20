package datastore

import (
	"fmt"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"

	r "gopkg.in/rethinkdb/rethinkdb-go.v6"
)

// IntegerPoolType defines the name of the IntegerPool
type IntegerPoolType string

func (p IntegerPoolType) String() string {
	return string(p)
}

const (
	// VRFIntegerPool defines the name of the pool for VRFs
	// this also defines the name of the tables
	// FIXME, must be renamed to vrfpool later
	VRFIntegerPool IntegerPoolType = "integerpool"
	// ASNIntegerPool defines the name of the pool for ASNs
	ASNIntegerPool IntegerPoolType = "asnpool"
)

var (
	// VRFPoolRangeMin the minimum integer to get from the vrf pool
	VRFPoolRangeMin = uint(1)
	// VRFPoolRangeMax the maximum integer to get from the vrf pool
	VRFPoolRangeMax = uint(131072)
	// ASNPoolRangeMin the minimum integer to get from the asn pool
	ASNPoolRangeMin = uint(1)
	// ASNPoolRangeMax the maximum integer to get from the asn pool
	ASNPoolRangeMax = uint(131072)
)

// IntegerPool manages unique integers
type IntegerPool struct {
	tablename string
	min       uint
	max       uint
	term      *r.Term
	session   r.QueryExecutor
}

type integer struct {
	ID uint `rethinkdb:"id" json:"id"`
}

// integerinfo contains information on the integer pool.
type integerinfo struct {
	IsInitialized bool `rethinkdb:"isInitialized" json:"isInitialized"`
}

// GetIntegerPool returns a named integerpool if already created
func (rs *RethinkStore) GetIntegerPool(pool IntegerPoolType) (*IntegerPool, error) {
	ip, ok := rs.integerPools[pool]
	if !ok {
		return nil, fmt.Errorf("no integerpool for %s created", pool)
	}
	return ip, nil
}

// initIntegerPool initializes a pool to acquire unique integers from.
// the acquired integers are used from the network service for defining the:
// one integer for:
// - vrf name
// - vni
// - vxlan-id
// and one integer for:
// - asn-id offset added to 4210000000 (ASNBase)
//
// the integer range has a vxlan-id constraint from Cumulus:
// 	net add vxlan vxlan10 vxlan id
//  <1-16777214>  :  An integer from 1 to 16777214
//
// in order not to impact performance too much, we limited the range of integers to 2^17=131072,
// which includes the range that we typically used for vrf names in the past.
// By this limitation we limit the number of machines possible to manage to ~130.000 !
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
func (rs *RethinkStore) initIntegerPool(pool IntegerPoolType, min, max uint) (*IntegerPool, error) {
	var result integerinfo
	tablename := pool.String()
	err := rs.findEntityByID(rs.integerInfoTable(tablename), &result, tablename)
	if err != nil {
		if !metal.IsNotFound(err) {
			return nil, err
		}
	}

	ip := &IntegerPool{
		tablename: tablename,
		min:       min,
		max:       max,
		session:   rs.session,
		term:      rs.integerTable(tablename),
	}
	rs.integerPools[pool] = ip
	rs.SugaredLogger.Infow("pool info", "table", tablename, "info", result)
	if result.IsInitialized {
		return ip, nil
	}

	rs.SugaredLogger.Infow("Initializing integer pool", "for", tablename, "RangeMin", min, "RangeMax", max)
	intRange := makeRange(min, max)
	_, err = rs.integerTable(tablename).Insert(intRange).RunWrite(rs.session, r.RunOpts{ArrayLimit: max})
	if err != nil {
		return nil, err
	}
	_, err = rs.integerInfoTable(tablename).Insert(map[string]interface{}{"id": tablename, "IsInitialized": true}).RunWrite(rs.session)
	return ip, err
}

func (ip *IntegerPool) RenewSession(term *r.Term, session r.QueryExecutor) {
	ip.term = term
	ip.session = session
}

// AcquireRandomUniqueInteger returns a random unique integer from the pool.
func (ip *IntegerPool) AcquireRandomUniqueInteger() (uint, error) {
	t := ip.term.Limit(1)
	return ip.genericAcquire(&t)
}

// AcquireUniqueInteger returns a unique integer from the pool.
func (ip *IntegerPool) AcquireUniqueInteger(value uint) (uint, error) {
	err := ip.verifyRange(value)
	if err != nil {
		return 0, err
	}
	t := ip.term.Get(value)
	return ip.genericAcquire(&t)
}

// ReleaseUniqueInteger returns a unique integer to the pool.
func (ip *IntegerPool) ReleaseUniqueInteger(id uint) error {
	err := ip.verifyRange(id)
	if err != nil {
		return err
	}

	i := integer{
		ID: id,
	}
	_, err = ip.term.Insert(i, r.InsertOpts{Conflict: "replace"}).RunWrite(ip.session)
	if err != nil {
		return err
	}

	return nil
}

func (ip *IntegerPool) genericAcquire(term *r.Term) (uint, error) {
	res, err := term.Delete(r.DeleteOpts{ReturnChanges: true}).RunWrite(ip.session)
	if err != nil {
		return 0, err
	}

	if len(res.Changes) == 0 {
		res, err := ip.term.Count().Run(ip.session)
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

func (ip *IntegerPool) verifyRange(value uint) error {
	if value < ip.min || value > ip.max {
		return fmt.Errorf("value '%d' is outside of the allowed range '%d - %d'", value, ip.min, ip.max)
	}
	return nil
}
