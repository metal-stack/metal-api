//go:build integration
// +build integration

package datastore

import (
	"context"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"github.com/metal-stack/metal-lib/pkg/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	r "gopkg.in/rethinkdb/rethinkdb-go.v6"
)

type machineTestable struct{}

func (_ *machineTestable) wipe() error {
	_, err := sharedDS.machineTable().Delete().RunWrite(sharedDS.session)
	return err
}

func (_ *machineTestable) create(m *metal.Machine) error { // nolint:unused
	if m.Allocation != nil {
		return sharedDS.createEntity(sharedDS.machineTable(), m)
	}
	return sharedDS.CreateMachine(m)
}

func (_ *machineTestable) delete(id string) error { // nolint:unused
	return sharedDS.DeleteMachine(&metal.Machine{Base: metal.Base{ID: id}})
}

func (_ *machineTestable) update(old *metal.Machine, mutateFn func(s *metal.Machine)) error { // nolint:unused
	mod := *old
	if mutateFn != nil {
		mutateFn(&mod)
	}

	return sharedDS.UpdateMachine(old, &mod)
}

func (_ *machineTestable) find(id string) (*metal.Machine, error) { // nolint:unused
	return sharedDS.FindMachineByID(id)
}

func (_ *machineTestable) list() ([]*metal.Machine, error) { // nolint:unused
	res, err := sharedDS.ListMachines()
	if err != nil {
		return nil, err
	}

	return derefSlice(res), nil
}

func (_ *machineTestable) search(q *MachineSearchQuery) ([]*metal.Machine, error) { // nolint:unused
	var res metal.Machines
	err := sharedDS.SearchMachines(q, &res)
	if err != nil {
		return nil, err
	}

	return derefSlice(res), nil
}

func (_ *machineTestable) defaultBody(m *metal.Machine) *metal.Machine {
	if m.Hardware.Nics == nil {
		m.Hardware.Nics = metal.Nics{}
	}
	for i := range m.Hardware.Nics {
		nic := m.Hardware.Nics[i]
		if nic.Neighbors == nil {
			nic.Neighbors = metal.Nics{}
		}
		for i2 := range nic.Neighbors {
			neigh := nic.Neighbors[i2]
			if neigh.Neighbors == nil {
				neigh.Neighbors = metal.Nics{}
			}
			nic.Neighbors[i2] = neigh
		}
		m.Hardware.Nics[i] = nic
	}
	if m.Hardware.Disks == nil {
		m.Hardware.Disks = []metal.BlockDevice{}
	}
	if m.Hardware.MetalCPUs == nil {
		m.Hardware.MetalCPUs = []metal.MetalCPU{}
	}
	if m.Hardware.MetalGPUs == nil {
		m.Hardware.MetalGPUs = []metal.MetalGPU{}
	}
	if m.Tags == nil {
		m.Tags = []string{}
	}
	if m.Allocation != nil {
		if m.Allocation.MachineNetworks == nil {
			m.Allocation.MachineNetworks = []*metal.MachineNetwork{}
		}
		if m.Allocation.SSHPubKeys == nil {
			m.Allocation.SSHPubKeys = []string{}
		}
		for i := range m.Allocation.MachineNetworks {
			n := m.Allocation.MachineNetworks[i]
			if n.Prefixes == nil {
				n.Prefixes = []string{}
			}
			if n.IPs == nil {
				n.IPs = []string{}
			}
			if n.DestinationPrefixes == nil {
				n.DestinationPrefixes = []string{}
			}
		}
	}
	return m
}

func TestRethinkStore_FindMachine(t *testing.T) {
	tt := &machineTestable{}
	defer func() {
		require.NoError(t, tt.wipe())
	}()

	tests := []findTest[*metal.Machine, *MachineSearchQuery]{
		{
			name: "find",
			id:   "2",

			mock: []*metal.Machine{
				{Base: metal.Base{ID: "1"}},
				{Base: metal.Base{ID: "2"}},
				{Base: metal.Base{ID: "3"}},
			},
			want:    tt.defaultBody(&metal.Machine{Base: metal.Base{ID: "2"}}),
			wantErr: nil,
		},
		{
			name:    "not found",
			id:      "4",
			want:    nil,
			wantErr: metal.NotFound(`no machine with id "4" found`),
		},
	}
	for i := range tests {
		tests[i].run(t, tt)
	}
}

func TestRethinkStore_SearchMachines(t *testing.T) {
	tt := &machineTestable{}
	defer func() {
		require.NoError(t, tt.wipe())
	}()

	tests := []searchTest[*metal.Machine, *MachineSearchQuery]{
		{
			name: "empty result",
			q: &MachineSearchQuery{
				ID: pointer.Pointer("2"),
			},
			mock: []*metal.Machine{
				{Base: metal.Base{ID: "1"}},
			},
			want:    nil,
			wantErr: nil,
		},
		{
			name: "search by id",
			q: &MachineSearchQuery{
				ID: pointer.Pointer("2"),
			},
			mock: []*metal.Machine{
				{Base: metal.Base{ID: "1"}},
				{Base: metal.Base{ID: "2"}},
				{Base: metal.Base{ID: "3"}},
			},
			want: []*metal.Machine{
				tt.defaultBody(&metal.Machine{Base: metal.Base{ID: "2"}}),
			},
			wantErr: nil,
		},
		{
			name: "search by name",
			q: &MachineSearchQuery{
				Name: pointer.Pointer("b"),
			},
			mock: []*metal.Machine{
				{Base: metal.Base{ID: "1", Name: "a"}},
				{Base: metal.Base{ID: "2", Name: "b"}},
				{Base: metal.Base{ID: "3", Name: "c"}},
			},
			want: []*metal.Machine{
				tt.defaultBody(&metal.Machine{Base: metal.Base{ID: "2", Name: "b"}}),
			},
			wantErr: nil,
		},
		{
			name: "search by partition",
			q: &MachineSearchQuery{
				PartitionID: pointer.Pointer("b"),
			},
			mock: []*metal.Machine{
				{Base: metal.Base{ID: "1"}, PartitionID: "a"},
				{Base: metal.Base{ID: "2"}, PartitionID: "b"},
				{Base: metal.Base{ID: "3"}, PartitionID: "c"},
			},
			want: []*metal.Machine{
				tt.defaultBody(&metal.Machine{Base: metal.Base{ID: "2"}, PartitionID: "b"}),
			},
			wantErr: nil,
		},
		{
			name: "search by size",
			q: &MachineSearchQuery{
				SizeID: pointer.Pointer("b"),
			},
			mock: []*metal.Machine{
				{Base: metal.Base{ID: "1"}, SizeID: "a"},
				{Base: metal.Base{ID: "2"}, SizeID: "b"},
				{Base: metal.Base{ID: "3"}, SizeID: "b"},
			},
			want: []*metal.Machine{
				tt.defaultBody(&metal.Machine{Base: metal.Base{ID: "2"}, SizeID: "b"}),
				tt.defaultBody(&metal.Machine{Base: metal.Base{ID: "3"}, SizeID: "b"}),
			},
			wantErr: nil,
		},
		{
			name: "search by rack",
			q: &MachineSearchQuery{
				RackID: pointer.Pointer("b"),
			},
			mock: []*metal.Machine{
				{Base: metal.Base{ID: "1"}, RackID: "a"},
				{Base: metal.Base{ID: "2"}, RackID: "b"},
				{Base: metal.Base{ID: "3"}, RackID: "b"},
			},
			want: []*metal.Machine{
				tt.defaultBody(&metal.Machine{Base: metal.Base{ID: "2"}, RackID: "b"}),
				tt.defaultBody(&metal.Machine{Base: metal.Base{ID: "3"}, RackID: "b"}),
			},
			wantErr: nil,
		},
		{
			name: "search by tags",
			q: &MachineSearchQuery{
				Tags: []string{"a=b"},
			},
			mock: []*metal.Machine{
				{Base: metal.Base{ID: "1"}, Tags: []string{}},
				{Base: metal.Base{ID: "2"}, Tags: []string{"a=b"}},
				{Base: metal.Base{ID: "3"}, Tags: []string{"b=c"}},
			},
			want: []*metal.Machine{
				tt.defaultBody(&metal.Machine{Base: metal.Base{ID: "2"}, Tags: []string{"a=b"}}),
			},
			wantErr: nil,
		},
		{
			name: "search by allocation name",
			q: &MachineSearchQuery{
				AllocationName: pointer.Pointer("b-name"),
			},
			mock: []*metal.Machine{
				{Base: metal.Base{ID: "1"}, Allocation: &metal.MachineAllocation{Name: "a-name"}},
				{Base: metal.Base{ID: "2"}, Allocation: &metal.MachineAllocation{Name: "b-name"}},
				{Base: metal.Base{ID: "3"}, Allocation: &metal.MachineAllocation{Name: "c-name"}},
			},
			want: []*metal.Machine{
				tt.defaultBody(&metal.Machine{Base: metal.Base{ID: "2"}, Allocation: &metal.MachineAllocation{Name: "b-name"}}),
			},
			wantErr: nil,
		},
		{
			name: "search by allocation project",
			q: &MachineSearchQuery{
				AllocationProject: pointer.Pointer("a-project"),
			},
			mock: []*metal.Machine{
				{Base: metal.Base{ID: "1"}, Allocation: &metal.MachineAllocation{Project: "a-project"}},
				{Base: metal.Base{ID: "2"}, Allocation: &metal.MachineAllocation{Project: "b-project"}},
				{Base: metal.Base{ID: "3"}, Allocation: &metal.MachineAllocation{Project: "c-project"}},
			},
			want: []*metal.Machine{
				tt.defaultBody(&metal.Machine{Base: metal.Base{ID: "1"}, Allocation: &metal.MachineAllocation{Project: "a-project"}}),
			},
			wantErr: nil,
		},
		{
			name: "search by allocation image id",
			q: &MachineSearchQuery{
				AllocationImageID: pointer.Pointer("ubuntu"),
			},
			mock: []*metal.Machine{
				{Base: metal.Base{ID: "1"}, Allocation: &metal.MachineAllocation{ImageID: "ubuntu"}},
				{Base: metal.Base{ID: "2"}, Allocation: &metal.MachineAllocation{ImageID: "debian"}},
				{Base: metal.Base{ID: "3"}, Allocation: &metal.MachineAllocation{ImageID: "ubuntu"}},
			},
			want: []*metal.Machine{
				tt.defaultBody(&metal.Machine{Base: metal.Base{ID: "1"}, Allocation: &metal.MachineAllocation{ImageID: "ubuntu"}}),
				tt.defaultBody(&metal.Machine{Base: metal.Base{ID: "3"}, Allocation: &metal.MachineAllocation{ImageID: "ubuntu"}}),
			},
			wantErr: nil,
		},
		{
			name: "search by allocation hostname",
			q: &MachineSearchQuery{
				AllocationHostname: pointer.Pointer("host-c"),
			},
			mock: []*metal.Machine{
				{Base: metal.Base{ID: "1"}, Allocation: &metal.MachineAllocation{Hostname: "host-a"}},
				{Base: metal.Base{ID: "2"}, Allocation: &metal.MachineAllocation{Hostname: "host-b"}},
				{Base: metal.Base{ID: "3"}, Allocation: &metal.MachineAllocation{Hostname: "host-c"}},
			},
			want: []*metal.Machine{
				tt.defaultBody(&metal.Machine{Base: metal.Base{ID: "3"}, Allocation: &metal.MachineAllocation{Hostname: "host-c"}}),
			},
			wantErr: nil,
		},
		{
			name: "search by allocation role",
			q: &MachineSearchQuery{
				AllocationRole: pointer.Pointer(metal.RoleMachine),
			},
			mock: []*metal.Machine{
				{Base: metal.Base{ID: "1"}, Allocation: &metal.MachineAllocation{Role: metal.RoleFirewall}},
				{Base: metal.Base{ID: "2"}, Allocation: &metal.MachineAllocation{Role: metal.RoleMachine}},
				{Base: metal.Base{ID: "3"}},
				{Base: metal.Base{ID: "4"}, Allocation: &metal.MachineAllocation{Role: metal.RoleFirewall}},
				{Base: metal.Base{ID: "5"}, Allocation: &metal.MachineAllocation{Role: metal.RoleMachine}},
			},
			want: []*metal.Machine{
				tt.defaultBody(&metal.Machine{Base: metal.Base{ID: "2"}, Allocation: &metal.MachineAllocation{Role: metal.RoleMachine}}),
				tt.defaultBody(&metal.Machine{Base: metal.Base{ID: "5"}, Allocation: &metal.MachineAllocation{Role: metal.RoleMachine}}),
			},
			wantErr: nil,
		},
		{
			name: "search by allocation succeeded",
			q: &MachineSearchQuery{
				AllocationSucceeded: pointer.Pointer(true),
			},
			mock: []*metal.Machine{
				{Base: metal.Base{ID: "1"}, Allocation: &metal.MachineAllocation{Succeeded: false}},
				{Base: metal.Base{ID: "2"}, Allocation: &metal.MachineAllocation{Succeeded: true}},
				{Base: metal.Base{ID: "3"}},
			},
			want: []*metal.Machine{
				tt.defaultBody(&metal.Machine{Base: metal.Base{ID: "2"}, Allocation: &metal.MachineAllocation{Succeeded: true}}),
			},
			wantErr: nil,
		},
		{
			name: "search by network ids",
			q: &MachineSearchQuery{
				NetworkIDs: []string{"internet", "private-tenant-a"},
			},
			mock: []*metal.Machine{
				{Base: metal.Base{ID: "1"}},
				{Base: metal.Base{ID: "2"}, Allocation: &metal.MachineAllocation{MachineNetworks: []*metal.MachineNetwork{{NetworkID: "private-tenant-a"}, {NetworkID: "internet"}}}},
				{Base: metal.Base{ID: "3"}, Allocation: &metal.MachineAllocation{MachineNetworks: []*metal.MachineNetwork{{NetworkID: "internet"}}}},
			},
			want: []*metal.Machine{
				tt.defaultBody(&metal.Machine{Base: metal.Base{ID: "2"}, Allocation: &metal.MachineAllocation{MachineNetworks: []*metal.MachineNetwork{{NetworkID: "private-tenant-a"}, {NetworkID: "internet"}}}}),
			},
			wantErr: nil,
		},
		{
			name: "search by network prefixes",
			q: &MachineSearchQuery{
				NetworkPrefixes: []string{"192.168.1.0/24"},
			},
			mock: []*metal.Machine{
				{Base: metal.Base{ID: "1"}},
				{Base: metal.Base{ID: "2"}, Allocation: &metal.MachineAllocation{MachineNetworks: []*metal.MachineNetwork{{Prefixes: []string{"100.64.0.0/28"}}}}},
				{Base: metal.Base{ID: "3"}, Allocation: &metal.MachineAllocation{MachineNetworks: []*metal.MachineNetwork{{Prefixes: []string{"100.64.0.0/28"}}, {Prefixes: []string{"192.168.1.0/24"}}}}},
			},
			want: []*metal.Machine{
				tt.defaultBody(&metal.Machine{Base: metal.Base{ID: "3"}, Allocation: &metal.MachineAllocation{MachineNetworks: []*metal.MachineNetwork{{Prefixes: []string{"100.64.0.0/28"}}, {Prefixes: []string{"192.168.1.0/24"}}}}}),
			},
			wantErr: nil,
		},
		{
			name: "search by network prefixes 2",
			q: &MachineSearchQuery{
				NetworkPrefixes: []string{"100.64.0.0/28"},
			},
			mock: []*metal.Machine{
				{Base: metal.Base{ID: "1"}},
				{Base: metal.Base{ID: "2"}, Allocation: &metal.MachineAllocation{MachineNetworks: []*metal.MachineNetwork{{Prefixes: []string{"100.64.0.0/28"}}}}},
				{Base: metal.Base{ID: "3"}, Allocation: &metal.MachineAllocation{MachineNetworks: []*metal.MachineNetwork{{Prefixes: []string{"100.64.0.0/28"}}, {Prefixes: []string{"192.168.1.0/24"}}}}},
			},
			want: []*metal.Machine{
				tt.defaultBody(&metal.Machine{Base: metal.Base{ID: "2"}, Allocation: &metal.MachineAllocation{MachineNetworks: []*metal.MachineNetwork{{Prefixes: []string{"100.64.0.0/28"}}}}}),
				tt.defaultBody(&metal.Machine{Base: metal.Base{ID: "3"}, Allocation: &metal.MachineAllocation{MachineNetworks: []*metal.MachineNetwork{{Prefixes: []string{"100.64.0.0/28"}}, {Prefixes: []string{"192.168.1.0/24"}}}}}),
			},
			wantErr: nil,
		},
		{
			name: "search by network ips",
			q: &MachineSearchQuery{
				NetworkIPs: []string{"192.168.1.3"},
			},
			mock: []*metal.Machine{
				{Base: metal.Base{ID: "1"}},
				{Base: metal.Base{ID: "2"}, Allocation: &metal.MachineAllocation{MachineNetworks: []*metal.MachineNetwork{{IPs: []string{"192.168.1.0"}}}}},
				{Base: metal.Base{ID: "3"}, Allocation: &metal.MachineAllocation{MachineNetworks: []*metal.MachineNetwork{{IPs: []string{"192.168.1.1"}}, {IPs: []string{"192.168.1.2", "192.168.1.3"}}}}},
			},
			want: []*metal.Machine{
				tt.defaultBody(&metal.Machine{Base: metal.Base{ID: "3"}, Allocation: &metal.MachineAllocation{MachineNetworks: []*metal.MachineNetwork{{IPs: []string{"192.168.1.1"}}, {IPs: []string{"192.168.1.2", "192.168.1.3"}}}}}),
			},
			wantErr: nil,
		},
		{
			name: "search by network destination prefixes",
			q: &MachineSearchQuery{
				NetworkDestinationPrefixes: []string{"0.0.0.0/0"},
			},
			mock: []*metal.Machine{
				{Base: metal.Base{ID: "1"}},
				{Base: metal.Base{ID: "2"}, Allocation: &metal.MachineAllocation{MachineNetworks: []*metal.MachineNetwork{{DestinationPrefixes: []string{"192.168.1.0/24"}}}}},
				{Base: metal.Base{ID: "3"}, Allocation: &metal.MachineAllocation{MachineNetworks: []*metal.MachineNetwork{{DestinationPrefixes: []string{"0.0.0.0/0"}}, {DestinationPrefixes: []string{"192.168.1.0/24", "0.0.0.0/0"}}}}},
			},
			want: []*metal.Machine{
				tt.defaultBody(&metal.Machine{Base: metal.Base{ID: "3"}, Allocation: &metal.MachineAllocation{MachineNetworks: []*metal.MachineNetwork{{DestinationPrefixes: []string{"0.0.0.0/0"}}, {DestinationPrefixes: []string{"192.168.1.0/24", "0.0.0.0/0"}}}}}),
			},
			wantErr: nil,
		},
		{
			name: "search by network vrf",
			q: &MachineSearchQuery{
				NetworkVrfs: []int64{2},
			},
			mock: []*metal.Machine{
				{Base: metal.Base{ID: "1"}},
				{Base: metal.Base{ID: "2"}, Allocation: &metal.MachineAllocation{MachineNetworks: []*metal.MachineNetwork{{Vrf: 0}}}},
				{Base: metal.Base{ID: "3"}, Allocation: &metal.MachineAllocation{MachineNetworks: []*metal.MachineNetwork{{Vrf: 1}, {Vrf: 2}}}},
			},
			want: []*metal.Machine{
				tt.defaultBody(&metal.Machine{Base: metal.Base{ID: "3"}, Allocation: &metal.MachineAllocation{MachineNetworks: []*metal.MachineNetwork{{Vrf: 1}, {Vrf: 2}}}}),
			},
			wantErr: nil,
		},
		{
			name: "search by network asn",
			q: &MachineSearchQuery{
				NetworkASNs: []int64{42000, 42001},
			},
			mock: []*metal.Machine{
				{Base: metal.Base{ID: "1"}},
				{Base: metal.Base{ID: "2"}, Allocation: &metal.MachineAllocation{MachineNetworks: []*metal.MachineNetwork{{ASN: 42000}}}},
				{Base: metal.Base{ID: "3"}, Allocation: &metal.MachineAllocation{MachineNetworks: []*metal.MachineNetwork{{ASN: 42000}, {ASN: 42001}}}},
			},
			want: []*metal.Machine{
				tt.defaultBody(&metal.Machine{Base: metal.Base{ID: "3"}, Allocation: &metal.MachineAllocation{MachineNetworks: []*metal.MachineNetwork{{ASN: 42000}, {ASN: 42001}}}}),
			},
			wantErr: nil,
		},
		{
			name: "search by hardware memory",
			q: &MachineSearchQuery{
				HardwareMemory: pointer.Pointer(int64(1000)),
			},
			mock: []*metal.Machine{
				{Base: metal.Base{ID: "1"}, Hardware: metal.MachineHardware{Memory: 1000}},
				{Base: metal.Base{ID: "2"}, Hardware: metal.MachineHardware{Memory: 5000}},
				{Base: metal.Base{ID: "3"}, Hardware: metal.MachineHardware{Memory: 1000}},
			},
			want: []*metal.Machine{
				tt.defaultBody(&metal.Machine{Base: metal.Base{ID: "1"}, Hardware: metal.MachineHardware{Memory: 1000}}),
				tt.defaultBody(&metal.Machine{Base: metal.Base{ID: "3"}, Hardware: metal.MachineHardware{Memory: 1000}}),
			},
			wantErr: nil,
		},
		{
			name: "search by nic mac address",
			q: &MachineSearchQuery{
				NicsMacAddresses: []string{"mac-c"},
			},
			mock: []*metal.Machine{
				{Base: metal.Base{ID: "1"}, Hardware: metal.MachineHardware{Nics: metal.Nics{{MacAddress: "mac-a"}}}},
				{Base: metal.Base{ID: "2"}, Hardware: metal.MachineHardware{Nics: metal.Nics{{MacAddress: "mac-b"}, {MacAddress: "mac-c"}}}},
				{Base: metal.Base{ID: "3"}, Hardware: metal.MachineHardware{Nics: metal.Nics{{MacAddress: "mac-d"}}}},
			},
			want: []*metal.Machine{
				tt.defaultBody(&metal.Machine{Base: metal.Base{ID: "2"}, Hardware: metal.MachineHardware{Nics: metal.Nics{{MacAddress: "mac-b"}, {MacAddress: "mac-c"}}}}),
			},
			wantErr: nil,
		},
		{
			name: "search by nic name",
			q: &MachineSearchQuery{
				NicsNames: []string{"nic-2"},
			},
			mock: []*metal.Machine{
				{Base: metal.Base{ID: "1"}, Hardware: metal.MachineHardware{Nics: metal.Nics{{Name: "nic-1"}}}},
				{Base: metal.Base{ID: "2"}, Hardware: metal.MachineHardware{Nics: metal.Nics{{Name: "nic-2"}, {Name: "nic-3"}}}},
				{Base: metal.Base{ID: "3"}, Hardware: metal.MachineHardware{Nics: metal.Nics{{Name: "nic-4"}}}},
			},
			want: []*metal.Machine{
				tt.defaultBody(&metal.Machine{Base: metal.Base{ID: "2"}, Hardware: metal.MachineHardware{Nics: metal.Nics{{Name: "nic-2"}, {Name: "nic-3"}}}}),
			},
			wantErr: nil,
		},
		{
			name: "search by nic vrf",
			q: &MachineSearchQuery{
				NicsVrfs: []string{"vrf10"},
			},
			mock: []*metal.Machine{
				{Base: metal.Base{ID: "1"}, Hardware: metal.MachineHardware{Nics: metal.Nics{{Vrf: "vrf1"}}}},
				{Base: metal.Base{ID: "2"}, Hardware: metal.MachineHardware{Nics: metal.Nics{{Vrf: "vrf10"}, {Name: "vrf11"}}}},
				{Base: metal.Base{ID: "3"}, Hardware: metal.MachineHardware{Nics: metal.Nics{{Vrf: ""}}}},
			},
			want: []*metal.Machine{
				tt.defaultBody(&metal.Machine{Base: metal.Base{ID: "2"}, Hardware: metal.MachineHardware{Nics: metal.Nics{{Vrf: "vrf10"}, {Name: "vrf11"}}}}),
			},
			wantErr: nil,
		},
		{
			name: "search by nic neighbor mac address",
			q: &MachineSearchQuery{
				NicsNeighborMacAddresses: []string{"mac-c"},
			},
			mock: []*metal.Machine{
				{Base: metal.Base{ID: "1"}, Hardware: metal.MachineHardware{Nics: metal.Nics{{Neighbors: metal.Nics{{MacAddress: "mac-a"}}}}}},
				{Base: metal.Base{ID: "2"}, Hardware: metal.MachineHardware{Nics: metal.Nics{{Neighbors: metal.Nics{{MacAddress: "mac-b"}, {MacAddress: "mac-c"}}}}}},
				{Base: metal.Base{ID: "3"}, Hardware: metal.MachineHardware{Nics: metal.Nics{{Neighbors: metal.Nics{{MacAddress: "mac-d"}}}}}},
			},
			want: []*metal.Machine{
				tt.defaultBody(&metal.Machine{Base: metal.Base{ID: "2"}, Hardware: metal.MachineHardware{Nics: metal.Nics{{Neighbors: metal.Nics{{MacAddress: "mac-b"}, {MacAddress: "mac-c"}}}}}}),
			},
			wantErr: nil,
		},
		{
			name: "search by nic neighbor name",
			q: &MachineSearchQuery{
				NicsNeighborNames: []string{"nic-2"},
			},
			mock: []*metal.Machine{
				{Base: metal.Base{ID: "1"}, Hardware: metal.MachineHardware{Nics: metal.Nics{{Neighbors: metal.Nics{{Name: "nic-1"}}}}}},
				{Base: metal.Base{ID: "2"}, Hardware: metal.MachineHardware{Nics: metal.Nics{{Neighbors: metal.Nics{{Name: "nic-2"}, {Name: "nic-3"}}}}}},
				{Base: metal.Base{ID: "3"}, Hardware: metal.MachineHardware{Nics: metal.Nics{{Neighbors: metal.Nics{{Name: "nic-4"}}}}}},
			},
			want: []*metal.Machine{
				tt.defaultBody(&metal.Machine{Base: metal.Base{ID: "2"}, Hardware: metal.MachineHardware{Nics: metal.Nics{{Neighbors: metal.Nics{{Name: "nic-2"}, {Name: "nic-3"}}}}}}),
			},
			wantErr: nil,
		},
		{
			name: "search by nic neighbor vrf",
			q: &MachineSearchQuery{
				NicsNeighborVrfs: []string{"vrf10"},
			},
			mock: []*metal.Machine{
				{Base: metal.Base{ID: "1"}, Hardware: metal.MachineHardware{Nics: metal.Nics{{Neighbors: metal.Nics{{Vrf: "vrf1"}}}}}},
				{Base: metal.Base{ID: "2"}, Hardware: metal.MachineHardware{Nics: metal.Nics{{Neighbors: metal.Nics{{Vrf: "vrf10"}, {Vrf: "vrf11"}}}}}},
				{Base: metal.Base{ID: "3"}},
			},
			want: []*metal.Machine{
				tt.defaultBody(&metal.Machine{Base: metal.Base{ID: "2"}, Hardware: metal.MachineHardware{Nics: metal.Nics{{Neighbors: metal.Nics{{Vrf: "vrf10"}, {Vrf: "vrf11"}}}}}}),
			},
			wantErr: nil,
		},
		{
			name: "search by disk name",
			q: &MachineSearchQuery{
				DiskNames: []string{"/dev/nvme0n1"},
			},
			mock: []*metal.Machine{
				{Base: metal.Base{ID: "1"}, Hardware: metal.MachineHardware{Disks: []metal.BlockDevice{{Name: "/dev/sda"}}}},
				{Base: metal.Base{ID: "2"}, Hardware: metal.MachineHardware{Disks: []metal.BlockDevice{{Name: "/dev/nvme0n1"}, {Name: "/dev/nvme0n2"}}}},
				{Base: metal.Base{ID: "3"}, Hardware: metal.MachineHardware{Disks: []metal.BlockDevice{{Name: "/dev/nvme0n1"}, {Name: "/dev/nvme0n2"}}}},
			},
			want: []*metal.Machine{
				tt.defaultBody(&metal.Machine{Base: metal.Base{ID: "2"}, Hardware: metal.MachineHardware{Disks: []metal.BlockDevice{{Name: "/dev/nvme0n1"}, {Name: "/dev/nvme0n2"}}}}),
				tt.defaultBody(&metal.Machine{Base: metal.Base{ID: "3"}, Hardware: metal.MachineHardware{Disks: []metal.BlockDevice{{Name: "/dev/nvme0n1"}, {Name: "/dev/nvme0n2"}}}}),
			},
			wantErr: nil,
		},
		{
			name: "search by disk size",
			q: &MachineSearchQuery{
				DiskSizes: []int64{1000},
			},
			mock: []*metal.Machine{
				{Base: metal.Base{ID: "1"}, Hardware: metal.MachineHardware{Disks: []metal.BlockDevice{{Size: 500}}}},
				{Base: metal.Base{ID: "2"}, Hardware: metal.MachineHardware{Disks: []metal.BlockDevice{{Size: 500}, {Size: 1000}}}},
				{Base: metal.Base{ID: "3"}, Hardware: metal.MachineHardware{Disks: []metal.BlockDevice{{Size: 500}, {Size: 1000}}}},
			},
			want: []*metal.Machine{
				tt.defaultBody(&metal.Machine{Base: metal.Base{ID: "2"}, Hardware: metal.MachineHardware{Disks: []metal.BlockDevice{{Size: 500}, {Size: 1000}}}}),
				tt.defaultBody(&metal.Machine{Base: metal.Base{ID: "3"}, Hardware: metal.MachineHardware{Disks: []metal.BlockDevice{{Size: 500}, {Size: 1000}}}}),
			},
			wantErr: nil,
		},
		{
			name: "search by state value reserved",
			q: &MachineSearchQuery{
				StateValue: pointer.Pointer(string(metal.ReservedState)),
			},
			mock: []*metal.Machine{
				{Base: metal.Base{ID: "1"}, State: metal.MachineState{Value: metal.AvailableState}},
				{Base: metal.Base{ID: "2"}, State: metal.MachineState{Value: metal.ReservedState}},
				{Base: metal.Base{ID: "3"}, State: metal.MachineState{Value: metal.AvailableState}},
				{Base: metal.Base{ID: "4"}, State: metal.MachineState{Value: metal.LockedState}},
			},
			want: []*metal.Machine{
				tt.defaultBody(&metal.Machine{Base: metal.Base{ID: "2"}, State: metal.MachineState{Value: metal.ReservedState}}),
			},
			wantErr: nil,
		},
		{
			name: "search by state value available",
			q: &MachineSearchQuery{
				StateValue: pointer.Pointer(string(metal.AvailableState)),
			},
			mock: []*metal.Machine{
				{Base: metal.Base{ID: "1"}, State: metal.MachineState{Value: metal.AvailableState}},
				{Base: metal.Base{ID: "2"}, State: metal.MachineState{Value: metal.ReservedState}},
				{Base: metal.Base{ID: "3"}, State: metal.MachineState{Value: metal.AvailableState}},
				{Base: metal.Base{ID: "4"}, State: metal.MachineState{Value: metal.LockedState}},
			},
			want: []*metal.Machine{
				tt.defaultBody(&metal.Machine{Base: metal.Base{ID: "1"}, State: metal.MachineState{Value: metal.AvailableState}}),
				tt.defaultBody(&metal.Machine{Base: metal.Base{ID: "3"}, State: metal.MachineState{Value: metal.AvailableState}}),
			},
			wantErr: nil,
		},
		{
			name: "search by ipmi address",
			q: &MachineSearchQuery{
				IpmiAddress: pointer.Pointer("1.1.1.2"),
			},
			mock: []*metal.Machine{
				{Base: metal.Base{ID: "1"}, IPMI: metal.IPMI{Address: "1.1.1.1"}},
				{Base: metal.Base{ID: "2"}, IPMI: metal.IPMI{Address: "1.1.1.2"}},
				{Base: metal.Base{ID: "3"}, IPMI: metal.IPMI{Address: "1.1.1.3"}},
			},
			want: []*metal.Machine{
				tt.defaultBody(&metal.Machine{Base: metal.Base{ID: "2"}, IPMI: metal.IPMI{Address: "1.1.1.2"}}),
			},
			wantErr: nil,
		},
		{
			name: "search by ipmi mac",
			q: &MachineSearchQuery{
				IpmiMacAddress: pointer.Pointer("mac-b"),
			},
			mock: []*metal.Machine{
				{Base: metal.Base{ID: "1"}, IPMI: metal.IPMI{MacAddress: "mac-a"}},
				{Base: metal.Base{ID: "2"}, IPMI: metal.IPMI{MacAddress: "mac-b"}},
				{Base: metal.Base{ID: "3"}, IPMI: metal.IPMI{MacAddress: "mac-c"}},
			},
			want: []*metal.Machine{
				tt.defaultBody(&metal.Machine{Base: metal.Base{ID: "2"}, IPMI: metal.IPMI{MacAddress: "mac-b"}}),
			},
			wantErr: nil,
		},
		{
			name: "search by ipmi user",
			q: &MachineSearchQuery{
				IpmiUser: pointer.Pointer("metal"),
			},
			mock: []*metal.Machine{
				{Base: metal.Base{ID: "1"}, IPMI: metal.IPMI{User: "metal"}},
				{Base: metal.Base{ID: "2"}, IPMI: metal.IPMI{User: ""}},
				{Base: metal.Base{ID: "3"}, IPMI: metal.IPMI{User: "metal"}},
			},
			want: []*metal.Machine{
				tt.defaultBody(&metal.Machine{Base: metal.Base{ID: "1"}, IPMI: metal.IPMI{User: "metal"}}),
				tt.defaultBody(&metal.Machine{Base: metal.Base{ID: "3"}, IPMI: metal.IPMI{User: "metal"}}),
			},
			wantErr: nil,
		},
		{
			name: "search by ipmi interface",
			q: &MachineSearchQuery{
				IpmiInterface: pointer.Pointer("lanplus"),
			},
			mock: []*metal.Machine{
				{Base: metal.Base{ID: "1"}, IPMI: metal.IPMI{Interface: "lanplus"}},
				{Base: metal.Base{ID: "2"}, IPMI: metal.IPMI{Interface: "lanplus"}},
				{Base: metal.Base{ID: "3"}, IPMI: metal.IPMI{Interface: ""}},
			},
			want: []*metal.Machine{
				tt.defaultBody(&metal.Machine{Base: metal.Base{ID: "1"}, IPMI: metal.IPMI{Interface: "lanplus"}}),
				tt.defaultBody(&metal.Machine{Base: metal.Base{ID: "2"}, IPMI: metal.IPMI{Interface: "lanplus"}}),
			},
			wantErr: nil,
		},
		{
			name: "search by fru chassis part number",
			q: &MachineSearchQuery{
				FruChassisPartNumber: pointer.Pointer("b-number"),
			},
			mock: []*metal.Machine{
				{Base: metal.Base{ID: "1"}, IPMI: metal.IPMI{Fru: metal.Fru{ChassisPartNumber: "a-number"}}},
				{Base: metal.Base{ID: "2"}, IPMI: metal.IPMI{Fru: metal.Fru{ChassisPartNumber: "b-number"}}},
				{Base: metal.Base{ID: "3"}, IPMI: metal.IPMI{Fru: metal.Fru{ChassisPartNumber: "c-number"}}},
			},
			want: []*metal.Machine{
				tt.defaultBody(&metal.Machine{Base: metal.Base{ID: "2"}, IPMI: metal.IPMI{Fru: metal.Fru{ChassisPartNumber: "b-number"}}}),
			},
			wantErr: nil,
		},
		{
			name: "search by fru chassis part serial",
			q: &MachineSearchQuery{
				FruChassisPartSerial: pointer.Pointer("b-serial"),
			},
			mock: []*metal.Machine{
				{Base: metal.Base{ID: "1"}, IPMI: metal.IPMI{Fru: metal.Fru{ChassisPartSerial: "a-serial"}}},
				{Base: metal.Base{ID: "2"}, IPMI: metal.IPMI{Fru: metal.Fru{ChassisPartSerial: "b-serial"}}},
				{Base: metal.Base{ID: "3"}, IPMI: metal.IPMI{Fru: metal.Fru{ChassisPartSerial: "c-serial"}}},
			},
			want: []*metal.Machine{
				tt.defaultBody(&metal.Machine{Base: metal.Base{ID: "2"}, IPMI: metal.IPMI{Fru: metal.Fru{ChassisPartSerial: "b-serial"}}}),
			},
			wantErr: nil,
		},
		{
			name: "search by fru board mfg",
			q: &MachineSearchQuery{
				FruBoardMfg: pointer.Pointer("b"),
			},
			mock: []*metal.Machine{
				{Base: metal.Base{ID: "1"}, IPMI: metal.IPMI{Fru: metal.Fru{BoardMfg: "a"}}},
				{Base: metal.Base{ID: "2"}, IPMI: metal.IPMI{Fru: metal.Fru{BoardMfg: "b"}}},
				{Base: metal.Base{ID: "3"}, IPMI: metal.IPMI{Fru: metal.Fru{BoardMfg: "c"}}},
			},
			want: []*metal.Machine{
				tt.defaultBody(&metal.Machine{Base: metal.Base{ID: "2"}, IPMI: metal.IPMI{Fru: metal.Fru{BoardMfg: "b"}}}),
			},
			wantErr: nil,
		},
		{
			name: "search by fru board mfg serial",
			q: &MachineSearchQuery{
				FruBoardMfgSerial: pointer.Pointer("b"),
			},
			mock: []*metal.Machine{
				{Base: metal.Base{ID: "1"}, IPMI: metal.IPMI{Fru: metal.Fru{BoardMfgSerial: "a"}}},
				{Base: metal.Base{ID: "2"}, IPMI: metal.IPMI{Fru: metal.Fru{BoardMfgSerial: "b"}}},
				{Base: metal.Base{ID: "3"}, IPMI: metal.IPMI{Fru: metal.Fru{BoardMfgSerial: "c"}}},
			},
			want: []*metal.Machine{
				tt.defaultBody(&metal.Machine{Base: metal.Base{ID: "2"}, IPMI: metal.IPMI{Fru: metal.Fru{BoardMfgSerial: "b"}}}),
			},
			wantErr: nil,
		},
		{
			name: "search by fru board part number",
			q: &MachineSearchQuery{
				FruBoardPartNumber: pointer.Pointer("b-number"),
			},
			mock: []*metal.Machine{
				{Base: metal.Base{ID: "1"}, IPMI: metal.IPMI{Fru: metal.Fru{BoardPartNumber: "a-number"}}},
				{Base: metal.Base{ID: "2"}, IPMI: metal.IPMI{Fru: metal.Fru{BoardPartNumber: "b-number"}}},
				{Base: metal.Base{ID: "3"}, IPMI: metal.IPMI{Fru: metal.Fru{BoardPartNumber: "c-number"}}},
			},
			want: []*metal.Machine{
				tt.defaultBody(&metal.Machine{Base: metal.Base{ID: "2"}, IPMI: metal.IPMI{Fru: metal.Fru{BoardPartNumber: "b-number"}}}),
			},
			wantErr: nil,
		},
		{
			name: "search by fru product manufacturer",
			q: &MachineSearchQuery{
				FruProductManufacturer: pointer.Pointer("b"),
			},
			mock: []*metal.Machine{
				{Base: metal.Base{ID: "1"}, IPMI: metal.IPMI{Fru: metal.Fru{ProductManufacturer: "a"}}},
				{Base: metal.Base{ID: "2"}, IPMI: metal.IPMI{Fru: metal.Fru{ProductManufacturer: "b"}}},
				{Base: metal.Base{ID: "3"}, IPMI: metal.IPMI{Fru: metal.Fru{ProductManufacturer: "c"}}},
			},
			want: []*metal.Machine{
				tt.defaultBody(&metal.Machine{Base: metal.Base{ID: "2"}, IPMI: metal.IPMI{Fru: metal.Fru{ProductManufacturer: "b"}}}),
			},
			wantErr: nil,
		},
		{
			name: "search by fru product part number",
			q: &MachineSearchQuery{
				FruProductPartNumber: pointer.Pointer("b-number"),
			},
			mock: []*metal.Machine{
				{Base: metal.Base{ID: "1"}, IPMI: metal.IPMI{Fru: metal.Fru{ProductPartNumber: "a-number"}}},
				{Base: metal.Base{ID: "2"}, IPMI: metal.IPMI{Fru: metal.Fru{ProductPartNumber: "b-number"}}},
				{Base: metal.Base{ID: "3"}, IPMI: metal.IPMI{Fru: metal.Fru{ProductPartNumber: "c-number"}}},
			},
			want: []*metal.Machine{
				tt.defaultBody(&metal.Machine{Base: metal.Base{ID: "2"}, IPMI: metal.IPMI{Fru: metal.Fru{ProductPartNumber: "b-number"}}}),
			},
			wantErr: nil,
		},
		{
			name: "search by fru product part serial",
			q: &MachineSearchQuery{
				FruProductSerial: pointer.Pointer("b-serial"),
			},
			mock: []*metal.Machine{
				{Base: metal.Base{ID: "1"}, IPMI: metal.IPMI{Fru: metal.Fru{ProductSerial: "a-serial"}}},
				{Base: metal.Base{ID: "2"}, IPMI: metal.IPMI{Fru: metal.Fru{ProductSerial: "b-serial"}}},
				{Base: metal.Base{ID: "3"}, IPMI: metal.IPMI{Fru: metal.Fru{ProductSerial: "c-serial"}}},
			},
			want: []*metal.Machine{
				tt.defaultBody(&metal.Machine{Base: metal.Base{ID: "2"}, IPMI: metal.IPMI{Fru: metal.Fru{ProductSerial: "b-serial"}}}),
			},
			wantErr: nil,
		},
	}

	for i := range tests {
		tests[i].run(t, tt)
	}
}

func TestRethinkStore_ListMachines(t *testing.T) {
	tt := &machineTestable{}
	defer func() {
		require.NoError(t, tt.wipe())
	}()

	tests := []listTest[*metal.Machine, *MachineSearchQuery]{
		{
			name: "list",
			mock: []*metal.Machine{
				{Base: metal.Base{ID: "1"}},
				{Base: metal.Base{ID: "2"}},
				{Base: metal.Base{ID: "3"}},
			},
			want: []*metal.Machine{
				tt.defaultBody(&metal.Machine{Base: metal.Base{ID: "1"}}),
				tt.defaultBody(&metal.Machine{Base: metal.Base{ID: "2"}}),
				tt.defaultBody(&metal.Machine{Base: metal.Base{ID: "3"}}),
			},
		},
	}
	for i := range tests {
		tests[i].run(t, tt)
	}
}

func TestRethinkStore_CreateMachine(t *testing.T) {
	tt := &machineTestable{}
	defer func() {
		require.NoError(t, tt.wipe())
	}()

	tests := []createTest[*metal.Machine, *MachineSearchQuery]{
		{
			name:    "create",
			want:    tt.defaultBody(&metal.Machine{Base: metal.Base{ID: "1"}}),
			wantErr: nil,
		},
		{
			name: "already exists",
			mock: []*metal.Machine{
				{Base: metal.Base{ID: "1"}},
			},
			want:    tt.defaultBody(&metal.Machine{Base: metal.Base{ID: "1"}}),
			wantErr: metal.Conflict(`cannot create machine in database, entity already exists: 1`),
		},
	}
	for i := range tests {
		tests[i].run(t, tt)
	}
}

func TestRethinkStore_DeleteMachine(t *testing.T) {
	tt := &machineTestable{}
	defer func() {
		require.NoError(t, tt.wipe())
	}()

	tests := []deleteTest[*metal.Machine, *MachineSearchQuery]{
		{
			name: "delete",
			id:   "2",
			mock: []*metal.Machine{
				{Base: metal.Base{ID: "1"}},
				{Base: metal.Base{ID: "2"}},
				{Base: metal.Base{ID: "3"}},
			},
			want: []*metal.Machine{
				tt.defaultBody(&metal.Machine{Base: metal.Base{ID: "1"}}),
				tt.defaultBody(&metal.Machine{Base: metal.Base{ID: "3"}}),
			},
		},
		{
			name: "not exists results in noop",
			id:   "abc",
			mock: []*metal.Machine{
				{Base: metal.Base{ID: "1"}},
				{Base: metal.Base{ID: "2"}},
				{Base: metal.Base{ID: "3"}},
			},
			want: []*metal.Machine{
				tt.defaultBody(&metal.Machine{Base: metal.Base{ID: "1"}}),
				tt.defaultBody(&metal.Machine{Base: metal.Base{ID: "2"}}),
				tt.defaultBody(&metal.Machine{Base: metal.Base{ID: "3"}}),
			},
		},
	}
	for i := range tests {
		tests[i].run(t, tt)
	}
}

func TestRethinkStore_UpdateMachine(t *testing.T) {
	tt := &machineTestable{}
	defer func() {
		require.NoError(t, tt.wipe())
	}()

	tests := []updateTest[*metal.Machine, *MachineSearchQuery]{
		{
			name: "update",
			mock: []*metal.Machine{
				{Base: metal.Base{ID: "1"}},
				{Base: metal.Base{ID: "2"}},
				{Base: metal.Base{ID: "3"}},
			},
			mutateFn: func(s *metal.Machine) {
				s.Tags = []string{"a=b"}
			},
			want: &metal.Machine{
				Base:     metal.Base{ID: "1"},
				Hardware: metal.MachineHardware{Nics: metal.Nics{}, Disks: []metal.BlockDevice{}, MetalCPUs: []metal.MetalCPU{}, MetalGPUs: []metal.MetalGPU{}},
				Tags:     []string{"a=b"},
			},
		},
	}
	for i := range tests {
		tests[i].run(t, tt)
	}
}

func Test_FindWaitingMachine_NoConcurrentModificationErrors(t *testing.T) {

	var (
		root  = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
		wg    sync.WaitGroup
		size  = metal.Size{Base: metal.Base{ID: "1"}}
		count int
	)

	for _, initEntity := range []struct {
		entity metal.Entity
		table  *r.Term
	}{
		{
			table: sharedDS.machineTable(),
			entity: &metal.Machine{
				Base: metal.Base{
					ID: "1",
				},
				PartitionID: "partition",
				SizeID:      size.ID,
				State: metal.MachineState{
					Value: metal.AvailableState,
				},
				Waiting:      true,
				PreAllocated: false,
			},
		},
		{
			table: sharedDS.eventTable(),
			entity: &metal.ProvisioningEventContainer{
				Base: metal.Base{
					ID: "1",
				},
				Liveliness: metal.MachineLivelinessAlive,
			},
		},
	} {
		initEntity := initEntity

		err := sharedDS.createEntity(initEntity.table, initEntity.entity)
		require.NoError(t, err)

		defer func() {
			_, err := initEntity.table.Delete().RunWrite(sharedDS.session)
			require.NoError(t, err)
		}()
	}

	for i := 0; i < 100; i++ {
		i := i
		wg.Add(1)

		log := root.With("worker", i)

		go func() {
			defer wg.Done()

			for {
				machine, err := sharedDS.FindWaitingMachine(context.Background(), "project", "partition", size, nil)
				if err != nil {
					if metal.IsConflict(err) {
						t.Errorf("concurrent modification occurred, shared mutex is not working")
						break
					}

					if strings.Contains(err.Error(), "no machine available") {
						continue
					}

					if strings.Contains(err.Error(), "too many parallel") {
						time.Sleep(10 * time.Millisecond)
						continue
					}

					t.Errorf("unexpected error occurred: %s", err)
					continue
				}

				log.Debug("waiting machine found")

				newMachine := *machine
				newMachine.PreAllocated = false
				if newMachine.Name == "" {
					newMachine.Name = strconv.Itoa(0)
				}

				assert.Equal(t, strconv.Itoa(count), newMachine.Name, "concurrency occurred")
				count++
				newMachine.Name = strconv.Itoa(count)

				err = sharedDS.updateEntity(sharedDS.machineTable(), &newMachine, machine)
				if err != nil {
					log.Error("unable to toggle back pre-allocation flag", "error", err)
					t.Fail()
				}

				return
			}
		}()
	}

	wg.Wait()

	assert.Equal(t, 100, count)
}
