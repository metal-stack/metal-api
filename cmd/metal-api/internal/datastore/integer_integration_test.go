// +build integration

package datastore

import (
	"context"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"github.com/metal-stack/metal-api/test"
	"go.uber.org/zap/zaptest"

	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRethinkStore_AcquireRandomUniqueIntegerIntegration(t *testing.T) {
	container, c, err := test.StartRethink()
	require.NoError(t, err)
	defer func() {
		_ = container.Terminate(context.Background())
	}()

	rs := New(zaptest.NewLogger(t), c.IP+":"+c.Port, c.DB, c.User, c.Password)
	rs.VRFPoolRangeMin = 10000
	rs.VRFPoolRangeMax = 10010

	err = rs.Connect()
	require.NoError(t, err)
	err = rs.Initialize()
	require.NoError(t, err)

	pool := rs.GetVRFPool()
	got, err := pool.AcquireRandomUniqueInteger()
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, got, uint(rs.VRFPoolRangeMin))
	assert.LessOrEqual(t, got, uint(rs.VRFPoolRangeMax))
}

func TestRethinkStore_AcquireUniqueIntegerTwiceIntegration(t *testing.T) {
	container, c, err := test.StartRethink()
	require.NoError(t, err)
	defer func() {
		_ = container.Terminate(context.Background())
	}()

	rs := New(zaptest.NewLogger(t), c.IP+":"+c.Port, c.DB, c.User, c.Password)
	rs.VRFPoolRangeMin = 10000
	rs.VRFPoolRangeMax = 10010

	err = rs.Connect()
	require.NoError(t, err)
	err = rs.Initialize()
	require.NoError(t, err)

	pool := rs.GetVRFPool()
	got, err := pool.AcquireUniqueInteger(10000)
	require.NoError(t, err)
	assert.Equal(t, got, uint(10000))

	_, err = pool.AcquireUniqueInteger(10000)
	assert.True(t, metal.IsConflict(err))
}

func TestRethinkStore_AcquireUniqueIntegerPoolExhaustionIntegration(t *testing.T) {
	container, c, err := test.StartRethink()
	require.NoError(t, err)
	defer func() {
		_ = container.Terminate(context.Background())
	}()

	rs := New(zaptest.NewLogger(t), c.IP+":"+c.Port, c.DB, c.User, c.Password)
	rs.VRFPoolRangeMin = 10000
	rs.VRFPoolRangeMax = 10010

	err = rs.Connect()
	require.NoError(t, err)
	err = rs.Initialize()
	require.NoError(t, err)

	pool := rs.GetVRFPool()

	for i := rs.VRFPoolRangeMin; i <= rs.VRFPoolRangeMax; i++ {
		got, err := pool.AcquireRandomUniqueInteger()
		require.NoError(t, err)
		assert.GreaterOrEqual(t, got, uint(rs.VRFPoolRangeMin))
		assert.LessOrEqual(t, got, uint(rs.VRFPoolRangeMax))
	}

	_, err = pool.AcquireRandomUniqueInteger()
	assert.True(t, metal.IsInternal(err))
}
