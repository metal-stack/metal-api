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
	expiration := 10 * time.Second

	err := sharedDS.sharedMutex.lock(ctx, "test", expiration, newLockOptAcquireTimeout(10*time.Millisecond))
	require.NoError(t, err)

	err = sharedDS.sharedMutex.lock(ctx, "test", expiration, newLockOptAcquireTimeout(5*time.Millisecond))
	require.Error(t, err)
	require.ErrorContains(t, err, "unable to acquire mutex")

	err = sharedDS.sharedMutex.lock(ctx, "test2", expiration, newLockOptAcquireTimeout(10*time.Millisecond))
	require.NoError(t, err)

	err = sharedDS.sharedMutex.lock(ctx, "test", expiration, newLockOptAcquireTimeout(10*time.Millisecond))
	require.Error(t, err)
	require.ErrorContains(t, err, "unable to acquire mutex")

	sharedDS.sharedMutex.unlock(ctx, "test")

	err = sharedDS.sharedMutex.lock(ctx, "test2", expiration, newLockOptAcquireTimeout(10*time.Millisecond))
	require.Error(t, err)
	require.ErrorContains(t, err, "unable to acquire mutex")

	err = sharedDS.sharedMutex.lock(ctx, "test", expiration, newLockOptAcquireTimeout(10*time.Millisecond))
	require.NoError(t, err)
}

func Test_sharedMutex_acquireAfterRelease(t *testing.T) {
	defer mutexCleanup(t)
	ctx := context.Background()

	err := sharedDS.sharedMutex.lock(ctx, "test", 3*time.Second, newLockOptAcquireTimeout(10*time.Millisecond))
	require.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()

		err = sharedDS.sharedMutex.lock(ctx, "test", 1*time.Second, newLockOptAcquireTimeout(3*time.Second))
		assert.NoError(t, err)
	}()

	time.Sleep(1 * time.Second)

	sharedDS.sharedMutex.unlock(ctx, "test")

	wg.Wait()
}

func Test_sharedMutex_expires(t *testing.T) {
	defer mutexCleanup(t)
	ctx := context.Background()

	err := sharedDS.sharedMutex.lock(ctx, "test", 2*time.Second, newLockOptAcquireTimeout(10*time.Millisecond))
	require.NoError(t, err)

	err = sharedDS.sharedMutex.lock(ctx, "test", 2*time.Second, newLockOptAcquireTimeout(10*time.Millisecond))
	require.Error(t, err)
	require.ErrorContains(t, err, "unable to acquire mutex")

	done := make(chan bool)
	go func() {
		err = sharedDS.sharedMutex.lock(ctx, "test", 2*time.Second, newLockOptAcquireTimeout(2*sharedDS.sharedMutex.checkinterval))
		if err != nil {
			t.Errorf("mutex was not acquired: %s", err)
		}
		done <- true
	}()

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 2*sharedDS.sharedMutex.checkinterval)
	defer cancel()

	select {
	case <-done:
	case <-timeoutCtx.Done():
		t.Errorf("shared mutex has not expired")
	}
}

func Test_sharedMutex_stop(t *testing.T) {
	defer mutexCleanup(t)
	ctx, cancel := context.WithCancel(context.Background())

	mutex, err := newSharedMutex(context.Background(), slog.Default(), sharedDS.dbsession)
	require.NoError(t, err)

	done := make(chan bool)

	go func() {
		mutex.expireloop(ctx)
		done <- true
	}()

	cancel()

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	select {
	case <-done:
	case <-timeoutCtx.Done():
		t.Errorf("shared mutex expiration did not stop")
	}
}

func mutexCleanup(t *testing.T) {
	_, err := r.Table("sharedmutex").Delete().RunWrite(sharedDS.dbsession)
	require.NoError(t, err)
}
