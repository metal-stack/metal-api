//go:build integration
// +build integration

package service

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	v1 "github.com/metal-stack/metal-api/cmd/metal-api/internal/service/v1"
	grpcv1 "github.com/metal-stack/metal-api/pkg/api/v1"
	"github.com/metal-stack/metal-api/test"
	"github.com/metal-stack/metal-lib/bus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMachineAllocationIntegrationFullCycle(t *testing.T) {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	rethinkContainer, cd, err := test.StartRethink(t)
	require.NoError(t, err)
	nsqContainer, publisher, consumer := test.StartNsqd(t, log)

	defer func() {
		_ = rethinkContainer.Terminate(context.Background())
		_ = nsqContainer.Terminate(context.Background())
	}()

	ds := datastore.New(log, cd.IP+":"+cd.Port, cd.DB, cd.User, cd.Password)
	ds.VRFPoolRangeMax = 1000
	ds.ASNPoolRangeMax = 1000

	te := createTestEnvironment(t, log, ds, publisher, consumer)

	// Register a machine
	mrr := &grpcv1.BootServiceRegisterRequest{
		Uuid: "test-uuid",
		Bios: &grpcv1.MachineBIOS{
			Version: "a",
			Vendor:  "metal",
			Date:    "1970",
		},
		Hardware: &grpcv1.MachineHardware{
			Cpus: []*grpcv1.MachineCPU{
				{
					Model:   "Intel Xeon Silver",
					Cores:   8,
					Threads: 8,
				},
			}, Memory: 1500,
			Disks: []*grpcv1.MachineBlockDevice{
				{
					Name: "sda",
					Size: 2500,
				},
			},
			Nics: []*grpcv1.MachineNic{
				{
					Name: "eth0",
					Mac:  "aa:aa:aa:aa:aa:aa",
					Neighbors: []*grpcv1.MachineNic{
						{
							Name: "swp1",
							Mac:  "bb:aa:aa:aa:aa:aa",
						},
					},
				},
				{
					Name: "eth1",
					Mac:  "aa:aa:aa:aa:aa:aa",
					Neighbors: []*grpcv1.MachineNic{
						{
							Name: "swp1",
							Mac:  "aa:bb:aa:aa:aa:aa",
						},
					},
				},
			},
		},
	}

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	conn, err := grpc.NewClient(te.listener.Addr().String(), opts...)
	require.NoError(t, err)

	conn.Connect()

	c := grpcv1.NewBootServiceClient(conn)

	registeredMachine, err := c.Register(ctx, mrr)
	require.NoError(t, err)
	require.NotNil(t, registeredMachine)
	assert.Len(t, mrr.Hardware.Nics, 2)

	err = te.machineWait(te.listener, "test-uuid")
	require.NoError(t, err)

	// DB contains at least a machine which is allocatable
	machine := v1.MachineAllocateRequest{
		ImageID:     "test-image-1.0.0",
		PartitionID: "test-partition",
		ProjectID:   "test-project-1",
		SizeID:      "test-size",
		Networks: v1.MachineAllocationNetworks{
			{
				NetworkID: te.privateNetwork.ID,
			},
		},
	}

	var allocatedMachine v1.MachineResponse
	status := te.machineAllocate(t, machine, &allocatedMachine)
	require.Equal(t, http.StatusOK, status)
	require.NotNil(t, allocatedMachine)
	require.NotNil(t, allocatedMachine.Allocation)
	require.NotNil(t, allocatedMachine.Allocation.Image)
	assert.Equal(t, machine.ImageID, allocatedMachine.Allocation.Image.ID)
	assert.Equal(t, machine.ProjectID, allocatedMachine.Allocation.Project)
	assert.Equal(t, string(metal.RoleMachine), allocatedMachine.Allocation.Role)
	assert.Len(t, allocatedMachine.Allocation.MachineNetworks, 1)
	assert.Equal(t, allocatedMachine.Allocation.MachineNetworks[0].NetworkType, metal.PrivatePrimaryUnshared.String())
	assert.NotEmpty(t, allocatedMachine.Allocation.MachineNetworks[0].Vrf)
	assert.GreaterOrEqual(t, allocatedMachine.Allocation.MachineNetworks[0].Vrf, te.ds.VRFPoolRangeMin)
	assert.LessOrEqual(t, allocatedMachine.Allocation.MachineNetworks[0].Vrf, te.ds.VRFPoolRangeMax)
	assert.GreaterOrEqual(t, allocatedMachine.Allocation.MachineNetworks[0].ASN, int64(ASNBase))
	assert.Len(t, allocatedMachine.Allocation.MachineNetworks[0].IPs, 1)
	_, ipnet, _ := net.ParseCIDR(te.privateNetwork.Prefixes[0])
	ip := net.ParseIP(allocatedMachine.Allocation.MachineNetworks[0].IPs[0])
	assert.True(t, ipnet.Contains(ip), "%s must be within %s", ip, ipnet)

	// Free machine for next test
	status = te.machineFree(t, "test-uuid", &v1.MachineResponse{})
	require.Equal(t, http.StatusOK, status)

	err = te.machineWait(te.listener, "test-uuid")
	require.NoError(t, err)

	// DB contains at least a machine which is allocatable
	machine = v1.MachineAllocateRequest{
		ImageID:     "test-image-1.0.0",
		PartitionID: "test-partition",
		ProjectID:   "test-project-1",
		SizeID:      "test-size",
		Networks: v1.MachineAllocationNetworks{
			{
				NetworkID: te.privateNetwork.ID,
			},
		},
	}

	allocatedMachine = v1.MachineResponse{}
	status = te.machineAllocate(t, machine, &allocatedMachine)
	require.Equal(t, http.StatusOK, status)
	require.NotNil(t, allocatedMachine)
	require.NotNil(t, allocatedMachine.Allocation)
	require.NotNil(t, allocatedMachine.Allocation.Image)
	assert.Equal(t, machine.ImageID, allocatedMachine.Allocation.Image.ID)
	assert.Equal(t, machine.ProjectID, allocatedMachine.Allocation.Project)
	assert.Len(t, allocatedMachine.Allocation.MachineNetworks, 1)
	assert.Equal(t, allocatedMachine.Allocation.MachineNetworks[0].NetworkType, metal.PrivatePrimaryUnshared.String())
	assert.NotEmpty(t, allocatedMachine.Allocation.MachineNetworks[0].Vrf)
	assert.GreaterOrEqual(t, allocatedMachine.Allocation.MachineNetworks[0].Vrf, te.ds.VRFPoolRangeMin)
	assert.LessOrEqual(t, allocatedMachine.Allocation.MachineNetworks[0].Vrf, te.ds.VRFPoolRangeMax)
	assert.GreaterOrEqual(t, allocatedMachine.Allocation.MachineNetworks[0].ASN, int64(ASNBase))
	assert.Len(t, allocatedMachine.Allocation.MachineNetworks[0].IPs, 1)
	_, ipnet, _ = net.ParseCIDR(te.privateNetwork.Prefixes[0])
	ip = net.ParseIP(allocatedMachine.Allocation.MachineNetworks[0].IPs[0])
	assert.True(t, ipnet.Contains(ip), "%s must be within %s", ip, ipnet)

	// Check if allocated machine created a machine <-> switch connection
	var foundSwitch v1.SwitchResponse
	status = te.switchGet(t, "test-switch01", &foundSwitch)
	require.Equal(t, http.StatusOK, status)
	require.NotNil(t, foundSwitch)
	require.Equal(t, "test-switch01", foundSwitch.ID)

	require.Len(t, foundSwitch.Connections, 1)
	require.Equal(t, "swp1", foundSwitch.Connections[0].Nic.Name, "we expected exactly one connection from one allocated machine->switch.swp1")
	require.Equal(t, "bb:aa:aa:aa:aa:aa", foundSwitch.Connections[0].Nic.MacAddress)
	require.Equal(t, "test-uuid", foundSwitch.Connections[0].MachineID, "the allocated machine ID must be connected to swp1")

	require.Len(t, foundSwitch.Nics, 1)
	require.NotNil(t, foundSwitch.Nics[0].BGPFilter)
	require.Len(t, foundSwitch.Nics[0].BGPFilter.CIDRs, 1, "on this switch port, only the cidrs from the allocated machine are allowed.")
	require.Equal(t, allocatedMachine.Allocation.MachineNetworks[0].Prefixes[0], foundSwitch.Nics[0].BGPFilter.CIDRs[0], "exactly the prefixes of the allocated machine must be allowed on this switch port")
	require.Empty(t, foundSwitch.Nics[0].BGPFilter.VNIs, "to this switch port a machine with no evpn connections, so no vni filter")

	// Free machine for next test
	status = te.machineFree(t, "test-uuid", &v1.MachineResponse{})
	require.Equal(t, http.StatusOK, status)

	// Check on the switch that connections still exists, but filters are nil,
	// this ensures that the freeMachine call executed and reset the machine<->switch configuration items.
	status = te.switchGet(t, "test-switch01", &foundSwitch)
	require.Equal(t, http.StatusOK, status)
	require.NotNil(t, foundSwitch)
	require.Equal(t, "test-switch01", foundSwitch.ID)

	require.Len(t, foundSwitch.Connections, 1, "machine is free for further allocations, but still connected to this switch port")
	require.Equal(t, "swp1", foundSwitch.Connections[0].Nic.Name, "we expected exactly one connection from one allocated machine->switch.swp1")
	require.Equal(t, "bb:aa:aa:aa:aa:aa", foundSwitch.Connections[0].Nic.MacAddress)
	require.Equal(t, "test-uuid", foundSwitch.Connections[0].MachineID, "the allocated machine ID must be connected to swp1")

	require.Len(t, foundSwitch.Nics, 1)
	require.Nil(t, foundSwitch.Nics[0].BGPFilter, "no machine allocated anymore")
}

func BenchmarkMachineList(b *testing.B) {
	rethinkContainer, c, err := test.StartRethink(b)
	require.NoError(b, err)
	defer func() {
		err = rethinkContainer.Terminate(context.TODO())
		require.NoError(b, err)
	}()

	now := time.Now()
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	ds := datastore.New(log, c.IP+":"+c.Port, c.DB, c.User, c.Password)
	ds.VRFPoolRangeMax = 1000
	ds.ASNPoolRangeMax = 1000

	err = ds.Connect()
	require.NoError(b, err)
	err = ds.Initialize()
	require.NoError(b, err)

	refCount := 100
	machineCount := 1000

	for i := range refCount {
		base := metal.Base{ID: strconv.Itoa(i)}
		img := &metal.Image{
			Base: base,
		}
		err := ds.CreateImage(img)
		require.NoError(b, err)

		par := &metal.Partition{
			Base: base,
		}
		err = ds.CreatePartition(par)
		require.NoError(b, err)

		size := &metal.Size{
			Base: base,
		}
		err = ds.CreateSize(size)
		require.NoError(b, err)
	}

	for i := range machineCount {
		base := metal.Base{ID: uuid.NewString()}
		refID := strconv.Itoa(i % refCount)

		m := &metal.Machine{
			Base:        base,
			SizeID:      refID,
			PartitionID: refID,
		}
		err := ds.CreateMachine(m)
		require.NoError(b, err)

		allocM := *m
		allocM.Allocation = &metal.MachineAllocation{
			ImageID: refID,
		}
		err = ds.UpdateMachine(m, &allocM)
		require.NoError(b, err)

		err = ds.CreateProvisioningEventContainer(&metal.ProvisioningEventContainer{Base: base, LastEventTime: &now})
		require.NoError(b, err)
	}

	machineService, err := NewMachine(log, ds, &emptyPublisher{}, bus.DirectEndpoints(), nil, nil, nil, nil, 0, nil, metal.DisabledIPMISuperUser())
	require.NoError(b, err)

	b.ResetTimer()

	for range b.N {
		var machines []v1.MachineResponse
		code := webRequestGet(b, machineService, &testUserDirectory.admin, nil, "/v1/machine", &machines)

		require.Equal(b, http.StatusOK, code)
		require.Len(b, machines, machineCount)
		require.NotNil(b, machines[0].Partition)
		require.NotEmpty(b, machines[0].Partition.ID)
		require.NotNil(b, machines[0].Size)
		require.NotEmpty(b, machines[0].Size.ID)
		require.NotNil(b, machines[0].Allocation)
		require.NotNil(b, machines[0].Allocation.Image)
		require.NotEmpty(b, machines[0].Allocation.Image.ID)
		require.NotNil(b, machines[0].RecentProvisioningEvents.LastEventTime)
	}

	b.StopTimer()
}
