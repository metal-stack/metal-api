//go:build integration
// +build integration

package service

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	v1 "github.com/metal-stack/metal-api/cmd/metal-api/internal/service/v1"
	grpcv1 "github.com/metal-stack/metal-api/pkg/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestSwitchReplacementIntegration(t *testing.T) {
	te := createTestEnvironment(t)
	defer te.teardown()

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
							Name:     "swp1",
							Mac:      "bb:aa:aa:aa:aa:aa",
							Hostname: "test-switch01",
						},
					},
				},
				{
					Name: "eth1",
					Mac:  "aa:aa:aa:aa:aa:aa",
					Neighbors: []*grpcv1.MachineNic{
						{
							Name:     "swp1",
							Mac:      "aa:bb:aa:aa:aa:aa",
							Hostname: "test-switch02",
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

	port := 50005
	conn, err := grpc.NewClient(fmt.Sprintf("localhost:%d", port), opts...)
	require.NoError(t, err)

	c := grpcv1.NewBootServiceClient(conn)

	registeredMachine, err := c.Register(ctx, mrr)
	require.NoError(t, err)
	require.NotNil(t, registeredMachine)
	assert.Len(t, mrr.Hardware.Nics, 2)

	// replace first switch

	sur := v1.SwitchUpdateRequest{
		Common: v1.Common{
			Identifiable: v1.Identifiable{
				ID: "test-switch01",
			},
		},
		SwitchBase: v1.SwitchBase{
			Mode: string(metal.SwitchReplace),
		},
	}

	var res v1.SwitchResponse
	status := te.switchUpdate(t, sur, &res)
	require.Equal(t, http.StatusOK, status)
	require.Equal(t, string(metal.SwitchReplace), res.SwitchBase.Mode)

	srr := v1.SwitchRegisterRequest{
		Common: v1.Common{
			Identifiable: v1.Identifiable{
				ID: "test-switch01",
			},
		},
		Nics: []v1.SwitchNic{
			{
				MacAddress: "aa:aa:bb:aa:aa:aa",
				Name:       "Ethernet4",
			},
		},
		PartitionID: "test-partition",
		SwitchBase: v1.SwitchBase{
			RackID: "test-rack",
			OS:     &v1.SwitchOS{Vendor: metal.SwitchOSVendorSonic},
		},
	}

	status = te.switchRegister(t, srr, &res)
	require.Equal(t, http.StatusOK, status)

	status = te.switchGet(t, "test-switch01", &res)
	require.Equal(t, http.StatusOK, status)
	require.Len(t, res.Nics, 1)
	require.Equal(t, srr.Nics[0].Name, res.Nics[0].Name)
	require.Equal(t, srr.Nics[0].MacAddress, res.Nics[0].MacAddress)
	require.Equal(t, string(metal.SwitchOperational), res.Mode)
	require.Len(t, res.Connections, 1)
	require.Equal(t, "test-uuid", res.Connections[0].MachineID)
	require.Equal(t, "Ethernet4", res.Connections[0].Nic.Name)
	require.Equal(t, "aa:aa:bb:aa:aa:aa", res.Connections[0].Nic.MacAddress)

	var mres v1.MachineResponse
	status = te.machineGet(t, mrr.Uuid, &mres)
	require.Equal(t, http.StatusOK, status)
	require.Len(t, mres.Hardware.Nics, 2)

	var nic v1.MachineNic
	for _, n := range mres.Hardware.Nics {
		if n.Name == "eth0" {
			nic = n
		}
	}
	require.Equal(t, "eth0", nic.Name)
	require.Len(t, nic.Neighbors, 1)
	require.Equal(t, "aa:aa:bb:aa:aa:aa", nic.Neighbors[0].MacAddress)
	require.Equal(t, "Ethernet4", nic.Neighbors[0].Name)

	// replace second switch

	sur = v1.SwitchUpdateRequest{
		Common: v1.Common{
			Identifiable: v1.Identifiable{
				ID: "test-switch02",
			},
		},
		SwitchBase: v1.SwitchBase{
			Mode: string(metal.SwitchReplace),
		},
	}

	status = te.switchUpdate(t, sur, &res)
	require.Equal(t, http.StatusOK, status)
	require.Equal(t, string(metal.SwitchReplace), res.SwitchBase.Mode)

	srr = v1.SwitchRegisterRequest{
		Common: v1.Common{
			Identifiable: v1.Identifiable{
				ID: "test-switch02",
			},
		},
		Nics: []v1.SwitchNic{
			{
				MacAddress: "aa:aa:aa:bb:aa:aa",
				Name:       "Ethernet4",
			},
		},
		PartitionID: "test-partition",
		SwitchBase: v1.SwitchBase{
			RackID: "test-rack",
			OS:     &v1.SwitchOS{Vendor: metal.SwitchOSVendorSonic},
		},
	}

	status = te.switchRegister(t, srr, &res)
	require.Equal(t, http.StatusOK, status)

	status = te.switchGet(t, "test-switch02", &res)
	require.Equal(t, http.StatusOK, status)
	require.Len(t, res.Nics, 1)
	require.Equal(t, srr.Nics[0].Name, res.Nics[0].Name)
	require.Equal(t, srr.Nics[0].MacAddress, res.Nics[0].MacAddress)
	require.Equal(t, string(metal.SwitchOperational), res.Mode)
	require.Len(t, res.Connections, 1)
	require.Equal(t, mrr.Uuid, res.Connections[0].MachineID)
	require.Equal(t, "Ethernet4", res.Connections[0].Nic.Name)
	require.Equal(t, "aa:aa:aa:bb:aa:aa", res.Connections[0].Nic.MacAddress)

	status = te.machineGet(t, mrr.Uuid, &mres)
	require.Equal(t, http.StatusOK, status)
	require.Len(t, mres.Hardware.Nics, 2)

	for _, n := range mres.Hardware.Nics {
		if n.Name == "eth1" {
			nic = n
		}
	}
	require.Equal(t, "eth1", nic.Name)
	require.Len(t, nic.Neighbors, 1)
	require.Equal(t, "aa:aa:aa:bb:aa:aa", nic.Neighbors[0].MacAddress)
	require.Equal(t, "Ethernet4", nic.Neighbors[0].Name)
}
