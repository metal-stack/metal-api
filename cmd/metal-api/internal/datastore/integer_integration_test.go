//go:build integration
// +build integration

package datastore

import (
	"context"
	"log/slog"
	"os"
	"sync"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"github.com/metal-stack/metal-api/test"

	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRethinkStore_AcquireRandomUniqueIntegerIntegration(t *testing.T) {
	container, c, err := test.StartRethink(t)
	require.NoError(t, err)
	defer func() {
		_ = container.Terminate(context.Background())
	}()

	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	rs := New(log, c.IP+":"+c.Port, c.DB, c.User, c.Password)
	rs.VRFPoolRangeMin = 10000
	rs.VRFPoolRangeMax = 10010
	rs.ASNPoolRangeMin = 10000
	rs.ASNPoolRangeMax = 10010

	err = rs.Connect()
	require.NoError(t, err)
	err = rs.Initialize()
	require.NoError(t, err)

	pool := rs.GetVRFPool()
	got, err := pool.AcquireRandomUniqueInteger()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, got, uint(rs.VRFPoolRangeMin))
	assert.LessOrEqual(t, got, uint(rs.VRFPoolRangeMax))
}

func TestRethinkStore_AcquireUniqueIntegerTwiceIntegration(t *testing.T) {
	container, c, err := test.StartRethink(t)
	require.NoError(t, err)
	defer func() {
		_ = container.Terminate(context.Background())
	}()
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	rs := New(log, c.IP+":"+c.Port, c.DB, c.User, c.Password)
	rs.VRFPoolRangeMin = 10000
	rs.VRFPoolRangeMax = 10010
	rs.ASNPoolRangeMin = 10000
	rs.ASNPoolRangeMax = 10010

	err = rs.Connect()
	require.NoError(t, err)
	err = rs.Initialize()
	require.NoError(t, err)

	pool := rs.GetVRFPool()
	got, err := pool.AcquireUniqueInteger(10000)
	require.NoError(t, err)
	assert.Equal(t, uint(10000), got)

	_, err = pool.AcquireUniqueInteger(10000)
	assert.True(t, metal.IsConflict(err))
}

func TestRethinkStore_AcquireUniqueIntegerPoolExhaustionIntegration(t *testing.T) {
	container, c, err := test.StartRethink(t)
	require.NoError(t, err)
	defer func() {
		_ = container.Terminate(context.Background())
	}()

	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	rs := New(log, c.IP+":"+c.Port, c.DB, c.User, c.Password)
	rs.VRFPoolRangeMin = 10000
	rs.VRFPoolRangeMax = 10010
	rs.ASNPoolRangeMin = 10000
	rs.ASNPoolRangeMax = 10010

	err = rs.Connect()
	require.NoError(t, err)
	err = rs.Initialize()
	require.NoError(t, err)

	pool := rs.GetVRFPool()
	var wg sync.WaitGroup

	for i := rs.VRFPoolRangeMin; i <= rs.VRFPoolRangeMax; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got, err := pool.AcquireRandomUniqueInteger()
			if err != nil {
				t.Fail()
			}
			assert.GreaterOrEqual(t, got, uint(rs.VRFPoolRangeMin))
			assert.LessOrEqual(t, got, uint(rs.VRFPoolRangeMax))
		}()
	}

	wg.Wait()

	_, err = pool.AcquireRandomUniqueInteger()
	assert.True(t, metal.IsInternal(err))
}
