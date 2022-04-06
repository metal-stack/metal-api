package datastore

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"go.uber.org/zap"

	"github.com/avast/retry-go/v4"
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
	// DefaultVRFPoolRangeMin the default minimum integer to get from the vrf pool
	DefaultVRFPoolRangeMin = uint(1)
	// DefaultVRFPoolRangeMax the default maximum integer to get from the vrf pool
	DefaultVRFPoolRangeMax = uint(131072)
	// DefaultASNPoolRangeMin the default minimum integer to get from the asn pool
	DefaultASNPoolRangeMin = uint(1)
	// DefaultASNPoolRangeMax the default maximum integer to get from the asn pool
	DefaultASNPoolRangeMax = uint(131072)
)

// IntegerPool manages unique integers
type IntegerPool struct {
	poolType  IntegerPoolType
	min       uint
	max       uint
	poolTable *r.Term
	infoTable *r.Term
	session   r.QueryExecutor
}

type integer struct {
	ID uint `rethinkdb:"id" json:"id"`
}

// integerinfo contains information on the integer pool.
type integerinfo struct {
	ID            string `rethinkdb:"id"`
	IsInitialized bool   `rethinkdb:"isInitialized" json:"isInitialized"`
}

func (rs *RethinkStore) GetVRFPool() *IntegerPool {
	return &IntegerPool{
		poolType:  VRFIntegerPool,
		session:   rs.session,
		min:       rs.VRFPoolRangeMin,
		max:       rs.VRFPoolRangeMax,
		poolTable: rs.vrfTable(),
		infoTable: rs.vrfInfoTable(),
	}
}

func (rs *RethinkStore) GetASNPool() *IntegerPool {
	return &IntegerPool{
		poolType:  ASNIntegerPool,
		session:   rs.session,
		min:       rs.ASNPoolRangeMin,
		max:       rs.ASNPoolRangeMax,
		poolTable: rs.asnTable(),
		infoTable: rs.asnInfoTable(),
	}
}

func (ip *IntegerPool) String() string {
	return ip.poolType.String()
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
func (ip *IntegerPool) initIntegerPool(log *zap.SugaredLogger) error {
	var info integerinfo
	err := ip.infoTable.ReadOne(&info, ip.session)
	if err != nil && !errors.Is(err, r.ErrEmptyResult) {
		return err
	}

	log.Infow("pool info", "id", ip.String(), "info", info)
	if info.IsInitialized {
		return nil
	}

	log.Infow("initializing integer pool", "for", ip.String(), "RangeMin", ip.min, "RangeMax", ip.max)
	intRange := makeRange(ip.min, ip.max)
	_, err = ip.poolTable.Insert(intRange).RunWrite(ip.session, r.RunOpts{ArrayLimit: ip.max})
	if err != nil {
		return err
	}

	_, err = ip.infoTable.Insert(integerinfo{
		ID:            ip.String(),
		IsInitialized: true,
	}).RunWrite(ip.session, r.RunOpts{
		Context: context.Background(), // this can take a while, do not abort
	})
	if err != nil {
		return err
	}

	return nil
}

// AcquireRandomUniqueInteger returns a random unique integer from the pool.
func (ip *IntegerPool) AcquireRandomUniqueInteger(ctx context.Context) (uint, error) {
	t := ip.poolTable.Limit(1)

	var integer uint
	err := retry.Do(
		func() error {
			var err2 error
			integer, err2 = ip.genericAcquire(ctx, &t)
			return err2
		},
		retry.Attempts(10),
		retry.MaxDelay(100*time.Millisecond),
		retry.LastErrorOnly(true),
	)

	return integer, err
}

// AcquireUniqueInteger returns a unique integer from the pool.
func (ip *IntegerPool) AcquireUniqueInteger(ctx context.Context, value uint) (uint, error) {
	err := ip.verifyRange(value)
	if err != nil {
		return 0, err
	}
	t := ip.poolTable.Get(value)
	return ip.genericAcquire(ctx, &t)
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
	_, err = ip.poolTable.Insert(i, r.InsertOpts{Conflict: "replace"}).RunWrite(ip.session)
	if err != nil {
		return err
	}

	return nil
}

func (ip *IntegerPool) genericAcquire(ctx context.Context, term *r.Term) (uint, error) {
	res, err := term.Delete(r.DeleteOpts{ReturnChanges: true}).RunWrite(ip.session)
	if err != nil {
		return 0, err
	}

	if len(res.Changes) == 0 {
		res, err := ip.poolTable.Count().Run(ip.session, r.RunOpts{
			Context: ctx,
		})
		if err != nil {
			return 0, err
		}
		var count int64
		err = res.One(&count)
		if err != nil {
			return 0, err
		}

		if count <= 0 {
			return 0, metal.Internal(errors.New("acquisition of a value failed for exhausted pool"), "")
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
