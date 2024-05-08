//go:build integration
// +build integration

package datastore

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	r "gopkg.in/rethinkdb/rethinkdb-go.v6"
)

func Test_sharedMutex_reallyLocking(t *testing.T) {
	defer mutexCleanup(t)
	ctx := context.Background()

	err := sharedDS.dbMutex.lock(ctx, "test", newLockOptAcquireTimeout(10*time.Millisecond))
	require.NoError(t, err)

	err = sharedDS.dbMutex.lock(ctx, "test", newLockOptAcquireTimeout(5*time.Millisecond))
	require.Error(t, err)
	require.ErrorContains(t, err, "unable to acquire mutex")

	err = sharedDS.dbMutex.lock(ctx, "test2", newLockOptAcquireTimeout(10*time.Millisecond))
	require.NoError(t, err)

	err = sharedDS.dbMutex.lock(ctx, "test", newLockOptAcquireTimeout(10*time.Millisecond))
	require.Error(t, err)
	require.ErrorContains(t, err, "unable to acquire mutex")

	sharedDS.dbMutex.unlock(ctx, "test")

	err = sharedDS.dbMutex.lock(ctx, "test2", newLockOptAcquireTimeout(10*time.Millisecond))
	require.Error(t, err)
	require.ErrorContains(t, err, "unable to acquire mutex")

	err = sharedDS.dbMutex.lock(ctx, "test", newLockOptAcquireTimeout(10*time.Millisecond))
	require.NoError(t, err)
}

func Test_sharedMutex_acquireAfterRelease(t *testing.T) {
	defer mutexCleanup(t)
	ctx := context.Background()

	err := sharedDS.dbMutex.lock(ctx, "test")
	require.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()

		err = sharedDS.dbMutex.lock(ctx, "test")
		assert.NoError(t, err)
	}()

	time.Sleep(1 * time.Second)

	sharedDS.dbMutex.unlock(ctx, "test")

	wg.Wait()
}

func Test_sharedMutex_expires(t *testing.T) {
	defer mutexCleanup(t)
	ctx := context.Background()

	err := sharedDS.dbMutex.lock(ctx, "test")
	require.NoError(t, err)

	err = sharedDS.dbMutex.lock(ctx, "test")
	require.Error(t, err)

	time.Sleep(sharedDS.dbMutex.checkinterval + 100*time.Millisecond)

	err = sharedDS.dbMutex.lock(ctx, "test")
	require.NoError(t, err)
}

func Test_sharedMutex_stop(t *testing.T) {
	defer mutexCleanup(t)
	ctx, cancel := context.WithCancel(context.Background())

	mutex := newSharedMutex(context.Background(), slog.Default(), sharedDS.dbsession, 3*time.Second)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		mutex.expireloop(ctx)
		wg.Done()
	}()

	cancel()

	wg.Wait()
}

func mutexCleanup(t *testing.T) {
	_, err := r.Table("sharedmutex").Delete().RunWrite(sharedDS.dbsession)
	require.NoError(t, err)
}
