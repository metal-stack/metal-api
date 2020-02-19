// +build integration

package service

import (
	"net"
	"net/http"
	"testing"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	v1 "github.com/metal-stack/metal-api/cmd/metal-api/internal/service/v1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testPrivateSuperCidr = "192.168.0.0/20"

func TestMachineAllocationIntegration(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	te := createTestEnvironment(t)
	defer te.teardown()

	// Empty DB with empty alloc request
	machine := v1.MachineAllocateRequest{}

	// Register a machine
	mrr := v1.MachineRegisterRequest{
		UUID:        "test-uuid",
		PartitionID: "test-partition",
		RackID:      "test-rack",
		Hardware: v1.MachineHardwareExtended{
			v1.MachineHardwareBase{
				CPUCores: 8,
				Memory:   1500,
				Disks: []v1.MachineBlockDevice{
					{
						Name: "sda",
						Size: 2500,
					},
				},
			},
			v1.MachineNicsExtended{
				{
					v1.MachineNic{
						Name:       "eth0",
						MacAddress: "aa:aa:aa:aa:aa:aa",
					},
					v1.MachineNicsExtended{
						{
							v1.MachineNic{
								Name:       "swp1",
								MacAddress: "bb:aa:aa:aa:aa:aa",
							},
							v1.MachineNicsExtended{},
						},
					},
				},
			},
		},
	}

	var registeredMachine v1.MachineResponse
	status := te.machineRegister(t, mrr, &registeredMachine)
	require.Equal(http.StatusCreated, status)
	require.NotNil(registeredMachine)
	assert.Equal(mrr.PartitionID, registeredMachine.Partition.ID)
	assert.Equal(mrr.RackID, registeredMachine.RackID)
	assert.Equal("test-size", registeredMachine.Size.ID)
	assert.Len(mrr.Hardware.Nics, 1)
	assert.Equal(mrr.Hardware.Nics[0].MachineNic.MacAddress, registeredMachine.Hardware.Nics[0].MacAddress)

	go te.machineWait("test-uuid")

	// DB contains at least a machine which is allocatable
	machine = v1.MachineAllocateRequest{
		ImageID:     "test-image",
		PartitionID: "test-partition",
		ProjectID:   te.project.ID,
		SizeID:      "test-size",
	}

	var allocatedMachine v1.MachineResponse
	status = te.machineAllocate(t, machine, &allocatedMachine)
	require.Equal(http.StatusOK, status)
	require.NotNil(allocatedMachine)
	require.NotNil(allocatedMachine.Allocation)
	require.NotNil(allocatedMachine.Allocation.Image)
	assert.Equal(machine.ImageID, allocatedMachine.Allocation.Image.ID)
	assert.Equal(machine.ProjectID, allocatedMachine.Allocation.Project)
	assert.Len(allocatedMachine.Allocation.MachineNetworks, 1)
	assert.True(allocatedMachine.Allocation.MachineNetworks[0].Private)
	assert.NotEmpty(allocatedMachine.Allocation.MachineNetworks[0].Vrf)
	assert.GreaterOrEqual(allocatedMachine.Allocation.MachineNetworks[0].Vrf, datastore.IntegerPoolRangeMin)
	assert.LessOrEqual(allocatedMachine.Allocation.MachineNetworks[0].Vrf, datastore.IntegerPoolRangeMax)
	assert.GreaterOrEqual(allocatedMachine.Allocation.MachineNetworks[0].ASN, metal.ASNBase)
	assert.Len(allocatedMachine.Allocation.MachineNetworks[0].IPs, 1)
	_, ipnet, _ := net.ParseCIDR(testPrivateSuperCidr)
	ip := net.ParseIP(allocatedMachine.Allocation.MachineNetworks[0].IPs[0])
	assert.True(ipnet.Contains(ip), "%s must be within %s", ip, ipnet)

	// Free machine for next test
	status = te.machineFree(t, "test-uuid", &v1.MachineResponse{})
	require.Equal(http.StatusOK, status)

	go te.machineWait("test-uuid")

	// DB contains at least a machine which is allocatable
	machine = v1.MachineAllocateRequest{
		ImageID:     "test-image",
		PartitionID: "test-partition",
		ProjectID:   te.project.ID,
		SizeID:      "test-size",
		Networks: v1.MachineAllocationNetworks{
			{
				NetworkID: te.privateNetwork.ID,
			},
		},
	}

	allocatedMachine = v1.MachineResponse{}
	status = te.machineAllocate(t, machine, &allocatedMachine)
	require.Equal(http.StatusOK, status)
	require.NotNil(allocatedMachine)
	require.NotNil(allocatedMachine.Allocation)
	require.NotNil(allocatedMachine.Allocation.Image)
	assert.Equal(machine.ImageID, allocatedMachine.Allocation.Image.ID)
	assert.Equal(machine.ProjectID, allocatedMachine.Allocation.Project)
	assert.Len(allocatedMachine.Allocation.MachineNetworks, 1)
	assert.True(allocatedMachine.Allocation.MachineNetworks[0].Private)
	assert.NotEmpty(allocatedMachine.Allocation.MachineNetworks[0].Vrf)
	assert.GreaterOrEqual(allocatedMachine.Allocation.MachineNetworks[0].Vrf, datastore.IntegerPoolRangeMin)
	assert.LessOrEqual(allocatedMachine.Allocation.MachineNetworks[0].Vrf, datastore.IntegerPoolRangeMax)
	assert.GreaterOrEqual(allocatedMachine.Allocation.MachineNetworks[0].ASN, metal.ASNBase)
	assert.Len(allocatedMachine.Allocation.MachineNetworks[0].IPs, 1)
	_, ipnet, _ = net.ParseCIDR(te.privateNetwork.Prefixes[0])
	ip = net.ParseIP(allocatedMachine.Allocation.MachineNetworks[0].IPs[0])
	assert.True(ipnet.Contains(ip), "%s must be within %s", ip, ipnet)

	// Check if allocated machine created a machine <-> switch connection
	var foundSwitch v1.SwitchResponse
	status = te.switchGet(t, "test-switch01", &foundSwitch)
	require.Equal(http.StatusOK, status)
	require.NotNil(foundSwitch)
	require.Equal("test-switch01", foundSwitch.ID)

	require.Len(foundSwitch.Connections, 1)
	require.Equal("swp1", foundSwitch.Connections[0].Nic.Name, "we expected exactly one connection from one allocated machine->switch.swp1")
	require.Equal("bb:aa:aa:aa:aa:aa", foundSwitch.Connections[0].Nic.MacAddress)
	require.Equal("test-uuid", foundSwitch.Connections[0].MachineID, "the allocated machine ID must be connected to swp1")

	require.Len(foundSwitch.Nics, 1)
	require.NotNil(foundSwitch.Nics[0].BGPFilter)
	require.Len(foundSwitch.Nics[0].BGPFilter.CIDRs, 1, "on this switch port, only the cidrs from the allocated machine are allowed.")
	require.Equal(allocatedMachine.Allocation.MachineNetworks[0].Prefixes[0], foundSwitch.Nics[0].BGPFilter.CIDRs[0], "exactly the prefixes of the allocated machine must be allowed on this switch port")
	require.Empty(foundSwitch.Nics[0].BGPFilter.VNIs, "to this switch port a machine with no evpn connections, so no vni filter")

	// Free machine for next test
	status = te.machineFree(t, "test-uuid", &v1.MachineResponse{})
	require.Equal(http.StatusOK, status)

	// Check on the switch that connections still exists, but filters are nil,
	// this ensures that the freeMachine call executed and reset the machine<->switch configuration items.
	status = te.switchGet(t, "test-switch01", &foundSwitch)
	require.Equal(http.StatusOK, status)
	require.NotNil(foundSwitch)
	require.Equal("test-switch01", foundSwitch.ID)

	require.Len(foundSwitch.Connections, 1, "machine is free for further allocations, but still connected to this switch port")
	require.Equal("swp1", foundSwitch.Connections[0].Nic.Name, "we expected exactly one connection from one allocated machine->switch.swp1")
	require.Equal("bb:aa:aa:aa:aa:aa", foundSwitch.Connections[0].Nic.MacAddress)
	require.Equal("test-uuid", foundSwitch.Connections[0].MachineID, "the allocated machine ID must be connected to swp1")

	require.Len(foundSwitch.Nics, 1)
	require.Nil(foundSwitch.Nics[0].BGPFilter, "no machine allocated anymore")

}
