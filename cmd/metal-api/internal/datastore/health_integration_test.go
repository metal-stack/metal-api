//go:build integration
// +build integration

package datastore

import (
	"context"
	"testing"

	"github.com/metal-stack/metal-lib/pkg/healthstatus"
	"github.com/stretchr/testify/require"
)

func TestRethinkStore_Health(t *testing.T) {
	result, err := sharedDS.Check(context.Background())
	require.NoError(t, err)
	require.Equal(t, healthstatus.HealthStatusHealthy, result.Status)
	require.Contains(t, result.Message, "connected to rethinkdb version: rethinkdb")
}
