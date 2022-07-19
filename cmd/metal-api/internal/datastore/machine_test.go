package datastore

import (
	"testing"
	"testing/quick"

	"github.com/google/go-cmp/cmp"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/testdata"
	r "gopkg.in/rethinkdb/rethinkdb-go.v6"
)

// Test that generates many input data
// Reference: https://golang.org/pkg/testing/quick/
func TestRethinkStore_FindMachineByID2(t *testing.T) {
	ds, mock := InitMockDB(t)
	testdata.InitMockDBData(mock)

	f := func(x string) bool {
		_, err := ds.FindMachineByID(x)
		returnvalue := true
		if err != nil {
			returnvalue = false
		}
		return returnvalue
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestRethinkStore_FindMachineByID(t *testing.T) {
	ds, mock := InitMockDB(t)
	testdata.InitMockDBData(mock)

	tests := []struct {
		name    string
		rs      *RethinkStore
		id      string
		want    *metal.Machine
		wantErr bool
	}{
		{
			name:    "Test 1",
			rs:      ds,
			id:      "1",
			want:    &testdata.M1,
			wantErr: false,
		},
		{
			name:    "Test 2",
			rs:      ds,
			id:      "2",
			want:    &testdata.M2,
			wantErr: false,
		},
		{
			name:    "Test 3",
			rs:      ds,
			id:      "404",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "Test 4",
			rs:      ds,
			id:      "999",
			want:    nil,
			wantErr: true,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.rs.FindMachineByID(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.FindMachine() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.want != nil {
				if diff := cmp.Diff(got, tt.want); diff != "" {
					t.Errorf("RethinkStore.FindMachine() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestRethinkStore_SearchMachine(t *testing.T) {
	ds, mock := InitMockDB(t)
	testdata.InitMockDBData(mock)

	mock.On(r.DB("mockdb").Table("machine").Filter(func(m r.Term) r.Term {
		return m.Field("hardware").Field("network_interfaces").Map(func(nic r.Term) r.Term {
			return nic.Field("macAddress")
		}).Contains(r.Expr("11:11:11"))
	})).Return(metal.Machines{
		testdata.M1,
	}, nil)

	tests := []struct {
		name    string
		rs      *RethinkStore
		mac     string
		want    metal.Machines
		wantErr bool
	}{
		// Test Data Array:
		{
			name: "Test 1",
			rs:   ds,
			mac:  "11:11:11",
			want: metal.Machines{
				testdata.M1,
			},
			wantErr: false,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			var got metal.Machines
			err := tt.rs.SearchMachines(&MachineSearchQuery{NicsMacAddresses: []string{tt.mac}}, &got)
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.FindMachines() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("RethinkStore.FindMachines() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRethinkStore_SearchMachine2(t *testing.T) {
	ds, mock := InitMockDB(t)
	testdata.InitMockDBData(mock)

	mock.On(r.DB("mockdb").Table("machine").Filter(func(m r.Term) r.Term {
		return m.Field("block_devices").Map(func(bd r.Term) r.Term {
			return bd.Field("size")
		}).Contains(r.Expr(int64(1000000000000)))
	})).Return(metal.Machines{
		testdata.M1,
	}, nil)

	tests := []struct {
		name    string
		rs      *RethinkStore
		size    int64
		want    metal.Machines
		wantErr bool
	}{
		{
			name: "Test 1",
			rs:   ds,
			size: 1000000000000,
			want: metal.Machines{
				testdata.M1,
			},
			wantErr: false,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			var got metal.Machines
			err := tt.rs.SearchMachines(&MachineSearchQuery{DiskSizes: []int64{tt.size}}, &got)
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.FindMachines() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("RethinkStore.FindMachines() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRethinkStore_SearchMachine3(t *testing.T) {
	ds, mock := InitMockDB(t)
	testdata.InitMockDBData(mock)

	mock.On(r.DB("mockdb").Table("machine").Filter(func(m r.Term) r.Term {
		return m.Field("allocation").Field("networks").Map(func(nw r.Term) r.Term {
			return nw.Field("networkid")
		}).Contains(r.Expr("1"))
	})).Return(metal.Machines{
		testdata.M1,
	}, nil)

	tests := []struct {
		name      string
		rs        *RethinkStore
		networkID string
		want      metal.Machines
		wantErr   bool
	}{
		// Test Data Array:
		{
			name:      "Test 1",
			rs:        ds,
			networkID: "1",
			want: metal.Machines{
				testdata.M1,
			},
			wantErr: false,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			var got metal.Machines
			err := tt.rs.SearchMachines(&MachineSearchQuery{NetworkIDs: []string{tt.networkID}}, &got)
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.FindMachines() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("RethinkStore.FindMachines() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRethinkStore_SearchMachine4(t *testing.T) {
	ds, mock := InitMockDB(t)
	testdata.InitMockDBData(mock)

	mock.On(r.DB("mockdb").Table("machine").Filter(func(m r.Term) r.Term {
		return m.Field("allocation").Field("networks").Map(func(nw r.Term) r.Term {
			return nw.Field("ips")
		}).Contains(r.Expr("1.2.3.4"))
	})).Return(metal.Machines{
		testdata.M1,
	}, nil)

	tests := []struct {
		name    string
		rs      *RethinkStore
		ip      string
		want    metal.Machines
		wantErr bool
	}{
		{
			name: "Test 1",
			rs:   ds,
			ip:   "1.2.3.4",
			want: metal.Machines{
				testdata.M1,
			},
			wantErr: false,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			var got metal.Machines
			err := tt.rs.SearchMachines(&MachineSearchQuery{NetworkIPs: []string{tt.ip}}, &got)
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.FindMachines() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("RethinkStore.FindMachines() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRethinkStore_SearchMachine5(t *testing.T) {
	ds, mock := InitMockDB(t)
	testdata.InitMockDBData(mock)

	mock.On(r.DB("mockdb").Table("machine").Filter(func(m r.Term) r.Term {
		return m.Field("allocation").Field("networks").Map(func(nw r.Term) r.Term {
			return nw.Field("prefixes")
		}).Contains(r.Expr("1.1.1.1/32"))
	})).Return(metal.Machines{
		testdata.M1,
	}, nil)

	tests := []struct {
		name    string
		rs      *RethinkStore
		prefix  string
		want    metal.Machines
		wantErr bool
	}{
		{
			name:   "Test 1",
			rs:     ds,
			prefix: "1.1.1.1/32",
			want: metal.Machines{
				testdata.M1,
			},
			wantErr: false,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			var got metal.Machines
			err := tt.rs.SearchMachines(&MachineSearchQuery{NetworkPrefixes: []string{tt.prefix}}, &got)
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.FindMachines() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("RethinkStore.FindMachines() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRethinkStore_SearchMachine6(t *testing.T) {
	ds, mock := InitMockDB(t)
	testdata.InitMockDBData(mock)

	mock.On(r.DB("mockdb").Table("machine").Filter(func(m r.Term) r.Term {
		return m.Field("hardware").Field("network_interfaces").Map(func(nic r.Term) r.Term {
			return nic.Field("neighbors").Map(func(neigh r.Term) r.Term {
				return neigh.Field("macAddress")
			})
		}).Contains(r.Expr("21:11:11:11:11:11"))
	})).Return(metal.Machines{
		testdata.M1,
	}, nil)

	tests := []struct {
		name    string
		rs      *RethinkStore
		mac     string
		want    metal.Machines
		wantErr bool
	}{
		{
			name: "Test 1",
			rs:   ds,
			mac:  "21:11:11:11:11:11",
			want: metal.Machines{
				testdata.M1,
			},
			wantErr: false,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			var got metal.Machines
			err := tt.rs.SearchMachines(&MachineSearchQuery{NicsNeighborMacAddresses: []string{tt.mac}}, &got)
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.FindMachines() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("RethinkStore.FindMachines() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRethinkStore_ListMachines(t *testing.T) {
	ds, mock := InitMockDB(t)
	testdata.InitMockDBData(mock)

	tests := []struct {
		name    string
		rs      *RethinkStore
		want    metal.Machines
		wantErr bool
	}{
		{
			name:    "Test 1",
			rs:      ds,
			want:    testdata.TestMachines,
			wantErr: false,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.rs.ListMachines()
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.ListMachines() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("RethinkStore.FindMachines() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRethinkStore_CreateMachine(t *testing.T) {
	ds, mock := InitMockDB(t)
	testdata.InitMockDBData(mock)

	tests := []struct {
		name    string
		rs      *RethinkStore
		machine *metal.Machine
		wantErr bool
	}{
		{
			name:    "Test 1",
			rs:      ds,
			machine: &testdata.M4,
			wantErr: false,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.rs.CreateMachine(tt.machine); (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.CreateMachine() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRethinkStore_DeleteMachine(t *testing.T) {
	ds, mock := InitMockDB(t)
	testdata.InitMockDBData(mock)

	tests := []struct {
		name    string
		rs      *RethinkStore
		machine *metal.Machine
		wantErr bool
	}{
		{
			name:    "Test 1",
			rs:      ds,
			machine: &testdata.M1,
			wantErr: false,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			err := tt.rs.DeleteMachine(tt.machine)
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.DeleteMachine() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestRethinkStore_UpdateMachine(t *testing.T) {
	ds, mock := InitMockDB(t)
	testdata.InitMockDBData(mock)

	tests := []struct {
		name       string
		rs         *RethinkStore
		oldMachine *metal.Machine
		newMachine *metal.Machine
		wantErr    bool
	}{
		{
			name:       "Test 1",
			rs:         ds,
			oldMachine: &testdata.M1,
			newMachine: &testdata.M2,
			wantErr:    false,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.rs.UpdateMachine(tt.oldMachine, tt.newMachine); (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.UpdateMachine() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TODO: Add tests for UpdateWaitingMachine, WaitForMachineAllocation, FindAvailableMachine
