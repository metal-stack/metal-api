//go:build integration
// +build integration

package datastore

import (
	"context"
	"os"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"github.com/metal-stack/metal-api/test"
	"github.com/testcontainers/testcontainers-go"
	"go.uber.org/zap"

	"testing"
)

// sharedDS is started before running the tests of this package (only once because it saves a lot of time then).
// it can be used for integration testing with a real rethinkdb.
//
// please make sure that after every test you clean up the data of the test in order not to have to deal with side-effects across
// the tests.
var sharedDS *RethinkStore

func TestMain(m *testing.M) {
	var container testcontainers.Container
	container, sharedDS = startRethinkInitialized()
	defer func() {
		err := container.Terminate(context.Background())
		panic(err)
	}()

	code := m.Run()
	os.Exit(code)
}

func startRethinkInitialized() (container testcontainers.Container, ds *RethinkStore) {
	container, c, err := test.StartRethink(nil)
	if err != nil {
		panic(err)
	}

	rs := New(zap.L().Sugar(), c.IP+":"+c.Port, c.DB, c.User, c.Password)
	rs.VRFPoolRangeMin = 10000
	rs.VRFPoolRangeMax = 10010
	rs.ASNPoolRangeMin = 10000
	rs.ASNPoolRangeMax = 10010

	err = rs.Connect()
	if err != nil {
		panic(err)
	}
	err = rs.Initialize()
	if err != nil {
		panic(err)
	}

	return container, rs
}

func ignoreTimestamps() cmp.Option {
	return cmpopts.IgnoreFields(metal.Base{}, "Created", "Changed")
}
