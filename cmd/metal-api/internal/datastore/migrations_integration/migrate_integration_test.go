//go:build integration
// +build integration

package migrations_integration

import (
	"context"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	_ "github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore/migrations"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"github.com/metal-stack/metal-api/test"
	"go.uber.org/zap/zaptest"

	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Migration(t *testing.T) {
	container, c, err := test.StartRethink(t)
	require.NoError(t, err)
	defer func() {
		_ = container.Terminate(context.Background())
	}()

	rs := datastore.New(zaptest.NewLogger(t).Sugar(), c.IP+":"+c.Port, c.DB, c.User, c.Password)
	rs.VRFPoolRangeMin = 10000
	rs.VRFPoolRangeMax = 10010
	rs.ASNPoolRangeMin = 10000
	rs.ASNPoolRangeMax = 10010

	err = rs.Connect()
	require.NoError(t, err)
	err = rs.Initialize()
	require.NoError(t, err)

	var (
		now           = time.Now()
		lastEventTime = now.Add(10 * time.Minute)
		ec            = &metal.ProvisioningEventContainer{
			Base: metal.Base{
				ID: "1",
			},
			Liveliness: "",
			Events: []metal.ProvisioningEvent{
				{
					Time:  now,
					Event: metal.ProvisioningEventPXEBooting,
				},
				{
					Time:  lastEventTime,
					Event: metal.ProvisioningEventPreparing,
				},
			},
			CrashLoop:            false,
			FailedMachineReclaim: false,
		}
		m = &metal.Machine{
			Base: metal.Base{
				ID: "1",
			},
		}
	)

	err = rs.UpsertProvisioningEventContainer(ec)
	require.NoError(t, err)

	err = rs.CreateMachine(m)
	require.NoError(t, err)

	updateM := *m
	updateM.Allocation = &metal.MachineAllocation{}
	err = rs.UpdateMachine(m, &updateM)
	require.NoError(t, err)

	err = rs.Migrate(nil, false)
	require.NoError(t, err)

	m, err = rs.FindMachineByID("1")
	require.NoError(t, err)

	assert.NotEmpty(t, m.Allocation.UUID, "allocation uuid was not generated")

	ec, err = rs.FindProvisioningEventContainer("1")
	require.NoError(t, err)
	require.NoError(t, ec.Validate())

	if diff := cmp.Diff(ec, &metal.ProvisioningEventContainer{
		Base: metal.Base{
			ID: "1",
		},
		Liveliness: "",
		Events: []metal.ProvisioningEvent{
			{
				Time:  lastEventTime,
				Event: metal.ProvisioningEventPreparing,
			},
			{
				Time:  now,
				Event: metal.ProvisioningEventPXEBooting,
			},
		},
		LastEventTime:        &lastEventTime,
		CrashLoop:            false,
		FailedMachineReclaim: false,
		// time comparison with time from rethink db is not possible due to different formats
	},
		cmpopts.IgnoreFields(metal.Base{}, "Changed"),
		cmpopts.IgnoreFields(metal.ProvisioningEvent{}, "Time"),
		cmpopts.IgnoreFields(metal.ProvisioningEventContainer{}, "LastEventTime"),
		cmpopts.IgnoreFields(metal.ProvisioningEventContainer{}, "Created"),
	); diff != "" {
		t.Errorf("RethinkStore.Migrate() mismatch (-want +got):\n%s", diff)
	}

	assert.Equal(t, ec.LastEventTime.Unix(), lastEventTime.Unix())
	assert.Equal(t, ec.Events[0].Time.Unix(), lastEventTime.Unix())
	assert.Equal(t, ec.Events[1].Time.Unix(), now.Unix())
}
