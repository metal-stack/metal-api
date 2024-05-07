//go:build integration
// +build integration

package datastore

import (
	"context"
	"log/slog"
	"os"
	"sort"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"github.com/metal-stack/metal-api/test"
	"github.com/metal-stack/metal-lib/pkg/testcommon"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"

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

	rs := New(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})), c.IP+":"+c.Port, c.DB, c.User, c.Password)
	rs.VRFPoolRangeMin = 10000
	rs.VRFPoolRangeMax = 10010
	rs.ASNPoolRangeMin = 10000
	rs.ASNPoolRangeMax = 10010
	rs.sharedMutexMaxBlockTime = 2 * time.Second

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

type testable[R metal.Entity, Q any] interface {
	wipe() error
	create(R) error
	find(string) (R, error)
	list() ([]R, error)
	search(Q) ([]R, error)
	delete(string) error
	update(R, func(R)) error
}

type findTest[R metal.Entity, Q any] struct {
	name    string
	id      string
	mock    []R
	want    R
	wantErr error
}

func (tt *findTest[R, Q]) run(t *testing.T, testable testable[R, Q]) {
	t.Helper()

	t.Run(tt.name, func(t *testing.T) {
		err := testable.wipe()
		require.NoError(t, err)

		for _, s := range tt.mock {
			s := s
			err := testable.create(s)
			require.NoError(t, err)
		}

		got, err := testable.find(tt.id)
		if diff := cmp.Diff(tt.wantErr, err, testcommon.ErrorStringComparer()); diff != "" {
			t.Errorf("error diff (-want +got):\n%s", diff)
		}

		if diff := cmp.Diff(tt.want, got, ignoreTimestamps()); diff != "" {
			t.Errorf("diff (-want +got):\n%s", diff)
		}
	})
}

type searchTest[R metal.Entity, Q any] struct {
	name    string
	q       Q
	mock    []R
	want    []R
	wantErr error
}

func (tt *searchTest[R, Q]) run(t *testing.T, testable testable[R, Q]) {
	t.Helper()

	t.Run(tt.name, func(t *testing.T) {
		err := testable.wipe()
		require.NoError(t, err)

		for _, s := range tt.mock {
			s := s
			err := testable.create(s)
			require.NoError(t, err)
		}

		got, err := testable.search(tt.q)
		if diff := cmp.Diff(tt.wantErr, err, testcommon.ErrorStringComparer()); diff != "" {
			t.Errorf("error diff (-want +got):\n%s", diff)
		}

		sort.Slice(got, func(i, j int) bool { return got[i].GetID() < got[j].GetID() })

		if diff := cmp.Diff(tt.want, got, ignoreTimestamps()); diff != "" {
			t.Errorf("diff (-want +got):\n%s", diff)
		}
	})
}

type listTest[R metal.Entity, Q any] struct {
	name    string
	mock    []R
	want    []R
	wantErr error
}

func (tt *listTest[R, Q]) run(t *testing.T, testable testable[R, Q]) {
	t.Helper()

	t.Run(tt.name, func(t *testing.T) {
		err := testable.wipe()
		require.NoError(t, err)

		for _, s := range tt.mock {
			s := s
			err := testable.create(s)
			require.NoError(t, err)
		}

		got, err := testable.list()
		if diff := cmp.Diff(tt.wantErr, err, testcommon.ErrorStringComparer()); diff != "" {
			t.Errorf("error diff (-want +got):\n%s", diff)
		}

		sort.Slice(got, func(i, j int) bool { return got[i].GetID() < got[j].GetID() })

		if diff := cmp.Diff(tt.want, got, ignoreTimestamps()); diff != "" {
			t.Errorf("diff (-want +got):\n%s", diff)
		}
	})
}

type createTest[R metal.Entity, Q any] struct {
	name    string
	mock    []R
	want    R
	wantErr error
}

func (tt *createTest[R, Q]) run(t *testing.T, testable testable[R, Q]) {
	t.Helper()

	t.Run(tt.name, func(t *testing.T) {
		err := testable.wipe()
		require.NoError(t, err)

		for _, s := range tt.mock {
			s := s
			err := testable.create(s)
			require.NoError(t, err)
		}

		err = testable.create(tt.want)
		if diff := cmp.Diff(tt.wantErr, err, testcommon.ErrorStringComparer()); diff != "" {
			t.Errorf("error diff (-want +got):\n%s", diff)
		}

		if tt.wantErr == nil {
			got, err := testable.find(tt.want.GetID())
			require.NoError(t, err)

			if diff := cmp.Diff(tt.want, got, ignoreTimestamps()); diff != "" {
				t.Errorf("diff (-want +got):\n%s", diff)
			}
		}
	})
}

type deleteTest[R metal.Entity, Q any] struct {
	name    string
	id      string
	mock    []R
	want    []R
	wantErr error
}

func (tt *deleteTest[R, Q]) run(t *testing.T, testable testable[R, Q]) {
	t.Helper()

	t.Run(tt.name, func(t *testing.T) {
		err := testable.wipe()
		require.NoError(t, err)

		for _, s := range tt.mock {
			s := s
			err := testable.create(s)
			require.NoError(t, err)
		}

		err = testable.delete(tt.id)
		if diff := cmp.Diff(tt.wantErr, err, testcommon.ErrorStringComparer()); diff != "" {
			t.Errorf("error diff (-want +got):\n%s", diff)
		}

		if tt.wantErr == nil {
			got, err := testable.list()
			require.NoError(t, err)

			sort.Slice(got, func(i, j int) bool { return got[i].GetID() < got[j].GetID() })

			if diff := cmp.Diff(tt.want, got, ignoreTimestamps()); diff != "" {
				t.Errorf("diff (-want +got):\n%s", diff)
			}
		}
	})
}

type updateTest[R metal.Entity, Q any] struct {
	name     string
	mock     []R
	mutateFn func(R)
	want     R
	wantErr  error
}

func (tt *updateTest[R, Q]) run(t *testing.T, testable testable[R, Q]) {
	t.Helper()

	t.Run(tt.name, func(t *testing.T) {
		err := testable.wipe()
		require.NoError(t, err)

		for _, s := range tt.mock {
			s := s
			err := testable.create(s)
			require.NoError(t, err)
		}

		old, err := testable.find(tt.want.GetID())
		require.NoError(t, err)

		err = testable.update(old, tt.mutateFn)
		if diff := cmp.Diff(tt.wantErr, err, testcommon.ErrorStringComparer()); diff != "" {
			t.Errorf("error diff (-want +got):\n%s", diff)
		}

		got, err := testable.find(tt.want.GetID())
		require.NoError(t, err)

		if diff := cmp.Diff(tt.want, got, ignoreTimestamps()); diff != "" {
			t.Errorf("diff (-want +got):\n%s", diff)
		}
	})
}

func derefSlice[R any](s []R) []*R { // nolint:unused
	var res []*R
	for _, e := range s {
		e := e
		res = append(res, &e)
	}
	return res
}
