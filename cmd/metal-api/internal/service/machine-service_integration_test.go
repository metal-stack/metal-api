//go:build integration
// +build integration

package service

import (
	"net"
	"net/http"
	"testing"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	v1 "github.com/metal-stack/metal-api/cmd/metal-api/internal/service/v1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMachineAllocationIntegrationFullCycle(t *testing.T) {
	te := createTestEnvironment(t)
	defer te.teardown()

	// Register a machine
	mrr := v1.MachineRegisterRequest{
		UUID:        "test-uuid",
		PartitionID: "test-partition",
		Hardware: v1.MachineHardware{
			MachineHardwareBase: v1.MachineHardwareBase{
				CPUCores: 8,
				Memory:   1500,
				Disks: []v1.MachineBlockDevice{
					{
						Name: "sda",
						Size: 2500,
					},
				},
			},
			Nics: v1.MachineNicsExtended{
				{
					MachineNic: v1.MachineNic{
						Name:       "eth0",
						MacAddress: "aa:aa:aa:aa:aa:aa",
					},
					Neighbors: v1.MachineNicsExtended{
						{
							MachineNic: v1.MachineNic{
								Name:       "swp1",
								MacAddress: "bb:aa:aa:aa:aa:aa",
							},
							Neighbors: v1.MachineNicsExtended{},
						},
					},
				},
				{
					MachineNic: v1.MachineNic{
						Name:       "eth1",
						MacAddress: "aa:aa:aa:aa:aa:aa",
					},
					Neighbors: v1.MachineNicsExtended{
						{
							MachineNic: v1.MachineNic{
								Name:       "swp1",
								MacAddress: "aa:bb:aa:aa:aa:aa",
							},
							Neighbors: v1.MachineNicsExtended{},
						},
					},
				},
			},
		},
	}

	var registeredMachine v1.MachineResponse
	status := te.machineRegister(t, mrr, &registeredMachine)
	require.Equal(t, http.StatusCreated, status)
	require.NotNil(t, registeredMachine)
	assert.Equal(t, mrr.PartitionID, registeredMachine.Partition.ID)
	assert.Equal(t, registeredMachine.RackID, "test-rack")
	assert.Len(t, mrr.Hardware.Nics, 2)
	assert.Equal(t, mrr.Hardware.Nics[0].MachineNic.MacAddress, registeredMachine.Hardware.Nics[0].MacAddress)

	err := te.machineWait("test-uuid")
	require.Nil(t, err)

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
	status = te.machineAllocate(t, machine, &allocatedMachine)
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

	err = te.machineWait("test-uuid")
	require.Nil(t, err)

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
