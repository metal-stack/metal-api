// +build integration

package datastore

import (
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"

	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRethinkStore_AcquireRandomUniqueIntegerIntegration(t *testing.T) {
	rs, c, ctx := InitTestDB(t)
	defer c.Terminate(ctx)
	got, err := rs.AcquireRandomUniqueInteger()
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, got, uint(IntegerPoolRangeMin))
	assert.LessOrEqual(t, got, uint(IntegerPoolRangeMax))
}

func TestRethinkStore_AcquireUniqueIntegerTwiceIntegration(t *testing.T) {
	rs, c, ctx := InitTestDB(t)
	defer c.Terminate(ctx)
	got, err := rs.AcquireUniqueInteger(10000)
	require.NoError(t, err)
	assert.Equal(t, got, uint(10000))

	_, err = rs.AcquireUniqueInteger(10000)
	assert.True(t, metal.IsConflict(err))
}

func TestRethinkStore_AcquireUniqueIntegerPoolExhaustionIntegration(t *testing.T) {
	rs, c, ctx := InitTestDB(t)
	defer c.Terminate(ctx)

	for i := IntegerPoolRangeMin; i <= IntegerPoolRangeMax; i++ {
		got, err := rs.AcquireRandomUniqueInteger()
		require.NoError(t, err)
		assert.GreaterOrEqual(t, got, uint(IntegerPoolRangeMin))
		assert.LessOrEqual(t, got, uint(IntegerPoolRangeMax))
	}

	_, err := rs.AcquireRandomUniqueInteger()
	assert.True(t, metal.IsInternal(err))
}
