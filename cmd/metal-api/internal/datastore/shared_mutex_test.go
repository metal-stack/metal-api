//go:build integration
// +build integration

package datastore

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	r "gopkg.in/rethinkdb/rethinkdb-go.v6"
)

func Test_sharedMutex_reallyLocking(t *testing.T) {
	defer mutexCleanup(t)
	ctx := context.Background()

	err := sharedDS.machineMutex.lock(ctx)
	require.NoError(t, err)

	err = sharedDS.machineMutex.lock(ctx)
	require.Error(t, err)
}

func Test_sharedMutex_acquireAfterRelease(t *testing.T) {
	defer mutexCleanup(t)
	ctx := context.Background()

	err := sharedDS.machineMutex.lock(ctx)
	require.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()

		err = sharedDS.machineMutex.lock(ctx)
		require.NoError(t, err)
	}()

	time.Sleep(1 * time.Second)

	err = sharedDS.machineMutex.unlock()
	require.NoError(t, err)

	wg.Wait()
}

func Test_sharedMutex_expires(t *testing.T) {
	defer mutexCleanup(t)
	ctx := context.Background()

	err := sharedDS.machineMutex.lock(ctx)
	require.NoError(t, err)

	err = sharedDS.machineMutex.lock(ctx)
	require.Error(t, err)

	time.Sleep(sharedDS.machineMutex.checkinterval + 100*time.Millisecond)

	err = sharedDS.machineMutex.lock(ctx)
	require.NoError(t, err)
}

func mutexCleanup(t *testing.T) {
	_, err := r.Table("sharedmutex").Delete().RunWrite(sharedDS.dbsession)
	require.NoError(t, err)
}
