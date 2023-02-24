package datastore

import (
	"reflect"
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

func Test_groupByRack(t *testing.T) {
	type args struct {
		machines metal.Machines
		rackids  []string
	}
	tests := []struct {
		name string
		args args
		want map[string]metal.Machines
	}{
		{
			name: "racks of size 1",
			args: args{
				machines: metal.Machines{
					{RackID: "1"},
					{RackID: "2"},
					{RackID: "3"},
				},
				rackids: []string{"1", "2", "3"},
			},
			want: map[string]metal.Machines{
				"1": {{RackID: "1"}},
				"2": {{RackID: "2"}},
				"3": {{RackID: "3"}},
			},
		},
		{
			name: "bigger racks",
			args: args{
				machines: metal.Machines{
					{RackID: "1"},
					{RackID: "2"},
					{RackID: "3"},
					{RackID: "1"},
					{RackID: "2"},
					{RackID: "3"},
					{RackID: "1"},
					{RackID: "2"},
					{RackID: "3"},
				},
				rackids: []string{"1", "2", "3"},
			},
			want: map[string]metal.Machines{
				"1": {
					{RackID: "1"},
					{RackID: "1"},
					{RackID: "1"},
				},
				"2": {
					{RackID: "2"},
					{RackID: "2"},
					{RackID: "2"},
				},
				"3": {
					{RackID: "3"},
					{RackID: "3"},
					{RackID: "3"},
				},
			},
		},
		{
			name: "empty racks",
			args: args{
				machines: metal.Machines{
					{RackID: "1"},
					{RackID: "2"},
					{RackID: "3"},
				},
				rackids: []string{"1", "2", "3", "4"},
			},
			want: map[string]metal.Machines{
				"1": {{RackID: "1"}},
				"2": {{RackID: "2"}},
				"3": {{RackID: "3"}},
				"4": {},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := groupByRack(tt.args.machines, tt.args.rackids); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("groupByRack() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_groupByTags(t *testing.T) {
	type args struct {
		machines metal.Machines
	}
	tests := []struct {
		name string
		args args
		want map[string]metal.Machines
	}{
		{
			name: "one machine with multiple tags",
			args: args{
				machines: metal.Machines{
					{Tags: []string{"1", "2", "3", "4"}},
				},
			},
			want: map[string]metal.Machines{
				"1": {
					{Tags: []string{"1", "2", "3", "4"}},
				},
				"2": {
					{Tags: []string{"1", "2", "3", "4"}},
				},
				"3": {
					{Tags: []string{"1", "2", "3", "4"}},
				},
				"4": {
					{Tags: []string{"1", "2", "3", "4"}},
				},
			},
		},
		{
			name: "multiple machines with intersecting tags",
			args: args{
				machines: metal.Machines{
					{Tags: []string{"1", "2", "3"}},
					{Tags: []string{"1", "2", "4"}},
				},
			},
			want: map[string]metal.Machines{
				"1": {
					{Tags: []string{"1", "2", "3"}},
					{Tags: []string{"1", "2", "4"}},
				},
				"2": {
					{Tags: []string{"1", "2", "3"}},
					{Tags: []string{"1", "2", "4"}},
				},
				"3": {
					{Tags: []string{"1", "2", "3"}},
				},
				"4": {
					{Tags: []string{"1", "2", "4"}},
				},
			},
		},
		{
			name: "multiple machines with disjunct tags",
			args: args{
				machines: metal.Machines{
					{Tags: []string{"1", "2", "3"}},
					{Tags: []string{"4", "5", "6"}},
				},
			},
			want: map[string]metal.Machines{
				"1": {
					{Tags: []string{"1", "2", "3"}},
				},
				"2": {
					{Tags: []string{"1", "2", "3"}},
				},
				"3": {
					{Tags: []string{"1", "2", "3"}},
				},
				"4": {
					{Tags: []string{"4", "5", "6"}},
				},
				"5": {
					{Tags: []string{"4", "5", "6"}},
				},
				"6": {
					{Tags: []string{"4", "5", "6"}},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := groupByTags(tt.args.machines); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("groupByTags() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_electRacks(t *testing.T) {
	type args struct {
		racks map[string]metal.Machines
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "no racks",
			args: args{
				racks: map[string]metal.Machines{},
			},
			want: []string{},
		},
		{
			name: "one winner",
			args: args{
				racks: map[string]metal.Machines{
					"1": {{}, {}, {}},
					"2": {{}, {}},
					"3": {{}},
					"4": {},
				},
			},
			want: []string{"4"},
		},
		{
			name: "two winners",
			args: args{
				racks: map[string]metal.Machines{
					"1": {{}, {}, {}},
					"2": {{}, {}, {}},
					"3": {{}},
					"4": {{}},
				},
			},
			want: []string{"3", "4"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := electRacks(tt.args.racks); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("electRacks() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_filter(t *testing.T) {
	type args struct {
		machines map[string]metal.Machines
		keys     []string
	}
	tests := []struct {
		name string
		args args
		want map[string]metal.Machines
	}{
		{
			name: "idempotent",
			args: args{
				machines: map[string]metal.Machines{
					"1": {{Tags: []string{"1"}}},
					"2": {{Tags: []string{"2"}}},
					"3": {{Tags: []string{"3"}}},
				},
				keys: []string{"1", "2", "3"},
			},
			want: map[string]metal.Machines{
				"1": {{Tags: []string{"1"}}},
				"2": {{Tags: []string{"2"}}},
				"3": {{Tags: []string{"3"}}},
			},
		},
		{
			name: "return empty",
			args: args{
				machines: map[string]metal.Machines{
					"1": {{Tags: []string{"1"}}},
					"2": {{Tags: []string{"2"}}},
					"3": {{Tags: []string{"3"}}},
				},
				keys: []string{"4", "5", "6"},
			},
			want: map[string]metal.Machines{},
		},
		{
			name: "return filtered",
			args: args{
				machines: map[string]metal.Machines{
					"1": {{Tags: []string{"1"}}},
					"2": {{Tags: []string{"2"}}},
					"3": {{Tags: []string{"3"}}},
				},
				keys: []string{"1", "5", "6"},
			},
			want: map[string]metal.Machines{
				"1": {{Tags: []string{"1"}}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := filter(tt.args.machines, tt.args.keys...); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("filter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_finalElection(t *testing.T) {
	type args struct {
		candidates [][]string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "no candidates",
			args: args{
				candidates: [][]string{},
			},
			want: []string{},
		},
		{
			name: "one winner",
			args: args{
				candidates: [][]string{
					{"1", "2", "3", "4"},
					{"1", "2", "3", "5"},
					{"1", "2", "4", "5"},
					{"1", "3", "4", "5"},
					{"1", "3", "4", "5"},
				},
			},
			want: []string{"1"},
		},
		{
			name: "two winners",
			args: args{
				candidates: [][]string{
					{"1", "2", "3", "4"},
					{"1", "2", "3", "5"},
					{"1", "3", "4", "5"},
					{"1", "3", "4", "5"},
					{"1", "3", "4", "5"},
				},
			},
			want: []string{"1", "3"},
		},
		{
			name: "all win",
			args: args{
				candidates: [][]string{
					{"1", "2", "3", "4"},
					{"1", "2", "3", "5"},
					{"1", "2", "4", "5"},
					{"1", "3", "4", "5"},
					{"2", "3", "4", "5"},
				},
			},
			want: []string{"1", "2", "3", "4", "5"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := finalElection(tt.args.candidates); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("finalElection() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_electMachine(t *testing.T) {
	type args struct {
		allMachines     metal.Machines
		projectMachines metal.Machines
		tags            []string
	}
	tests := []struct {
		name string
		args args
		want metal.Machine
	}{
		{
			name: "one available machine",
			args: args{
				allMachines:     metal.Machines{{RackID: "1"}},
				projectMachines: metal.Machines{},
				tags:            []string{},
			},
			want: metal.Machine{RackID: "1"},
		},
		{
			name: "no tags, spread by project only",
			args: args{
				allMachines: metal.Machines{
					{RackID: "1"},
					{RackID: "2"},
				},
				projectMachines: metal.Machines{
					{RackID: "1"},
				},
				tags: []string{},
			},
			want: metal.Machine{RackID: "2"},
		},
		{
			name: "spread by tags",
			args: args{
				allMachines: metal.Machines{
					{RackID: "1"},
					{RackID: "2"},
				},
				projectMachines: metal.Machines{
					{RackID: "1", Tags: []string{"tag1"}},
					{RackID: "2"},
					{RackID: "2"},
				},
				tags: []string{"tag1"},
			},
			want: metal.Machine{RackID: "2"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := electMachine(tt.args.allMachines, tt.args.projectMachines, tt.args.tags); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("electMachine() = %v, want %v", got, tt.want)
			}
		})
	}
}
