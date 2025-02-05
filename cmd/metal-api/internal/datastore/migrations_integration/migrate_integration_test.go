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

	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

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

func Test_MigrationChildPrefixLength(t *testing.T) {
	type tmpPartition struct {
		ID                         string `rethinkdb:"id"`
		PrivateNetworkPrefixLength uint8  `rethinkdb:"privatenetworkprefixlength"`
	}

	container, c, err := test.StartRethink(t)
	require.NoError(t, err)
	defer func() {
		_ = container.Terminate(context.Background())
	}()

	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	rs := datastore.New(log, c.IP+":"+c.Port, c.DB, c.User, c.Password)
	// limit poolsize to speed up initialization
	rs.VRFPoolRangeMin = 10000
	rs.VRFPoolRangeMax = 10010
	rs.ASNPoolRangeMin = 10000
	rs.ASNPoolRangeMax = 10010

	err = rs.Connect()
	require.NoError(t, err)
	err = rs.Initialize()
	require.NoError(t, err)

	var (
		p1 = &tmpPartition{
			ID:                         "p1",
			PrivateNetworkPrefixLength: 22,
		}
		p2 = &tmpPartition{
			ID:                         "p2",
			PrivateNetworkPrefixLength: 24,
		}
		p3 = &tmpPartition{
			ID: "p3",
		}
		n1 = &metal.Network{
			Base: metal.Base{
				ID: "n1",
			},
			PartitionID: "p1",
			Prefixes: metal.Prefixes{
				{IP: "10.0.0.0", Length: "8"},
			},
			PrivateSuper: true,
		}
		n2 = &metal.Network{
			Base: metal.Base{
				ID: "n2",
			},
			Prefixes: metal.Prefixes{
				{IP: "2001::", Length: "64"},
			},
			PartitionID:  "p2",
			PrivateSuper: true,
		}
		n3 = &metal.Network{
			Base: metal.Base{
				ID: "n3",
			},
			Prefixes: metal.Prefixes{
				{IP: "100.1.0.0", Length: "22"},
			},
			PartitionID:  "p2",
			PrivateSuper: false,
		}
		n4 = &metal.Network{
			Base: metal.Base{
				ID: "n4",
			},
			Prefixes: metal.Prefixes{
				{IP: "100.1.0.0", Length: "22"},
			},
			PartitionID:  "p3",
			PrivateSuper: true,
		}
	)
	_, err = r.DB("metal").Table("partition").Insert(p1).RunWrite(rs.Session())
	require.NoError(t, err)
	_, err = r.DB("metal").Table("partition").Insert(p2).RunWrite(rs.Session())
	require.NoError(t, err)
	_, err = r.DB("metal").Table("partition").Insert(p3).RunWrite(rs.Session())
	require.NoError(t, err)

	err = rs.CreateNetwork(n1)
	require.NoError(t, err)
	err = rs.CreateNetwork(n2)
	require.NoError(t, err)
	err = rs.CreateNetwork(n3)
	require.NoError(t, err)
	err = rs.CreateNetwork(n4)
	require.NoError(t, err)

	err = rs.Migrate(nil, false)
	require.NoError(t, err)

	p, err := rs.FindPartition(p1.ID)
	require.NoError(t, err)
	require.NotNil(t, p)
	p, err = rs.FindPartition(p2.ID)
	require.NoError(t, err)
	require.NotNil(t, p)

	n1fetched, err := rs.FindNetworkByID(n1.ID)
	require.NoError(t, err)
	require.NotNil(t, n1fetched)
	require.Equal(t, p1.PrivateNetworkPrefixLength, n1fetched.DefaultChildPrefixLength[metal.IPv4AddressFamily], "childprefixlength:%v", n1fetched.DefaultChildPrefixLength)
	require.Contains(t, n1fetched.AddressFamilies, metal.IPv4AddressFamily)

	n2fetched, err := rs.FindNetworkByID(n2.ID)
	require.NoError(t, err)
	require.NotNil(t, n2fetched)
	require.Equal(t, p2.PrivateNetworkPrefixLength, n2fetched.DefaultChildPrefixLength[metal.IPv4AddressFamily], "childprefixlength:%v", n2fetched.DefaultChildPrefixLength)
	require.Equal(t, n2fetched.DefaultChildPrefixLength, metal.ChildPrefixLength{metal.IPv4AddressFamily: 24, metal.IPv6AddressFamily: 64})
	require.Contains(t, n2fetched.AddressFamilies, metal.IPv6AddressFamily)

	n3fetched, err := rs.FindNetworkByID(n3.ID)
	require.NoError(t, err)
	require.NotNil(t, n3fetched)
	require.Nil(t, n3fetched.DefaultChildPrefixLength)
	require.Contains(t, n3fetched.AddressFamilies, metal.IPv4AddressFamily)

	n4fetched, err := rs.FindNetworkByID(n4.ID)
	require.NoError(t, err)
	require.NotNil(t, n4fetched)
	require.NotNil(t, n4fetched.DefaultChildPrefixLength)
	require.Contains(t, n4fetched.AddressFamilies, metal.IPv4AddressFamily)
	require.Equal(t, uint8(22), n4fetched.DefaultChildPrefixLength[metal.IPv4AddressFamily])
}
