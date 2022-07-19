//go:build integration
// +build integration

package datastore

import (
	"context"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"github.com/metal-stack/metal-api/test"
	"go.uber.org/zap/zaptest"

	"testing"

	"github.com/stretchr/testify/require"
)

func TestRethinkStore_ConflictIsReturned(t *testing.T) {
	container, c, err := test.StartRethink()
	require.NoError(t, err)
	defer func() {
		_ = container.Terminate(context.Background())
	}()

	rs := New(zaptest.NewLogger(t).Sugar(), c.IP+":"+c.Port, c.DB, c.User, c.Password)
	rs.VRFPoolRangeMin = 10000
	rs.VRFPoolRangeMax = 10010
	rs.ASNPoolRangeMin = 10000
	rs.ASNPoolRangeMax = 10010

	err = rs.Connect()
	require.NoError(t, err)
	err = rs.Initialize()
	require.NoError(t, err)

	err = rs.CreateImage(&metal.Image{Base: metal.Base{ID: "ubuntu"}})
	require.NoError(t, err)
	err = rs.CreateImage(&metal.Image{Base: metal.Base{ID: "debian"}})
	require.NoError(t, err)
	err = rs.CreateImage(&metal.Image{Base: metal.Base{ID: "ubuntu"}})
	require.EqualError(t, err, "Conflict cannot create image in database, entity already exists: ubuntu")

	genID := metal.Image{Base: metal.Base{ID: ""}}
	err = rs.CreateImage(&genID)
	require.NoError(t, err)
	require.NotEmpty(t, genID.ID)
}
