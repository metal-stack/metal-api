//go:build integration
// +build integration

package migrations_integration

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore/migrations"
	_ "github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore/migrations"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"github.com/metal-stack/metal-api/test"
	r "gopkg.in/rethinkdb/rethinkdb-go.v6"

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

	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	rs := datastore.New(log, c.IP+":"+c.Port, c.DB, c.User, c.Password)
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
		n = &metal.Network{
			Base: metal.Base{
				ID:   "tenant-super",
				Name: "tenant-super",
			},
			PrivateSuper: true,
		}
	)

	err = rs.UpsertProvisioningEventContainer(ec)
	require.NoError(t, err)

	err = rs.CreateMachine(m)
	require.NoError(t, err)

	err = rs.CreateNetwork(n)
	require.NoError(t, err)

	oldSize := migrations.OldSize_Mig07{
		Base: metal.Base{
			ID: "c1-xlarge-x86",
		},
		Reservations: []migrations.OldReservation_Mig07{
			{
				Amount:       3,
				Description:  "a description",
				ProjectID:    "project-1",
				PartitionIDs: []string{"partition-a"},
				Labels: map[string]string{
					"a": "b",
				},
			},
		},
	}

	_, err = r.DB("metal").Table("size").Insert(oldSize).RunWrite(rs.Session())
	require.NoError(t, err)

	updateM := *m
	updateM.Allocation = &metal.MachineAllocation{}
	err = rs.UpdateMachine(m, &updateM)
	require.NoError(t, err)

	// now run the migration
	err = rs.Migrate(nil, false)
	require.NoError(t, err)

	// assert
	m, err = rs.FindMachineByID("1")
	require.NoError(t, err)

	assert.NotEmpty(t, m.Allocation.UUID, "allocation uuid was not generated")

	n, err = rs.FindNetworkByID("tenant-super")
	require.NoError(t, err)

	assert.NotEmpty(t, n)
	assert.Equal(t, []string{"10.240.0.0/12"}, n.AdditionalAnnouncableCIDRs)

	rvs, err := rs.ListSizeReservations()
	require.NoError(t, err)

	require.Len(t, rvs, 1)
	require.NotEmpty(t, rvs[0].ID)
	if diff := cmp.Diff(rvs, metal.SizeReservations{
		{
			Base: metal.Base{
				Description: "a description",
			},
			SizeID:       "c1-xlarge-x86",
			Amount:       3,
			ProjectID:    "project-1",
			PartitionIDs: []string{"partition-a"},
			Labels: map[string]string{
				"a": "b",
			},
		},
	}, cmpopts.IgnoreFields(metal.SizeReservation{}, "ID", "Created", "Changed")); diff != "" {
		t.Errorf("size reservations diff: %s", diff)
	}

	sizes, err := rs.ListSizes()
	require.NoError(t, err)
	require.Len(t, sizes, 1)

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
