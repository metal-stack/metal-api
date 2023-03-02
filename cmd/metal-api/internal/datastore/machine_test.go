package datastore

import (
	"reflect"
	"testing"
	"testing/quick"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/testdata"
)

// Test that generates many input data
// Reference: https://golang.org/pkg/testing/quick/
func TestRethinkStore_FindMachineByIDQuick(t *testing.T) {
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
			name: "no machines",
			args: args{
				machines: metal.Machines{},
				rackids:  []string{"1", "2", "3"},
			},
			want: map[string]metal.Machines{
				"1": {},
				"2": {},
				"3": {},
			},
		},
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
			name: "one machine with no tags",
			args: args{
				machines: metal.Machines{{}},
			},
			want: map[string]metal.Machines{},
		},
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
			name: "empty map",
			args: args{
				machines: map[string]metal.Machines{},
				keys:     []string{"1", "2", "3"},
			},
			want: map[string]metal.Machines{},
		},
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
				projectMachines: metal.Machines{{RackID: "1", Tags: []string{"tag"}}},
				tags:            []string{"tag"},
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
					{RackID: "1"},
					{RackID: "2", Tags: []string{"tag"}},
				},
				tags: []string{},
			},
			want: metal.Machine{RackID: "2"},
		},
		{
			name: "no tags and preferred racks aren't available",
			args: args{
				allMachines: metal.Machines{
					{RackID: "1"},
				},
				projectMachines: metal.Machines{
					{RackID: "1"},
					{RackID: "1"},
					{RackID: "2"},
				},
				tags: []string{},
			},
			want: metal.Machine{RackID: "1"},
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
					{RackID: "2", Tags: []string{"irrelevant-tag"}},
					{RackID: "2"},
				},
				tags: []string{"tag1"},
			},
			want: metal.Machine{RackID: "2"},
		},
		{
			name: "no machines match relevant tags",
			args: args{
				allMachines: metal.Machines{
					{RackID: "1"},
					{RackID: "2"},
				},
				projectMachines: metal.Machines{
					{RackID: "1"},
					{RackID: "2", Tags: []string{"irrelevant-tag"}},
					{RackID: "2"},
				},
				tags: []string{"tag1"},
			},
			want: metal.Machine{RackID: "1"},
		},
		{
			name: "two racks in a draw, let project decide",
			args: args{
				allMachines: metal.Machines{
					{RackID: "1"},
					{RackID: "2"},
					{RackID: "3"},
				},
				projectMachines: metal.Machines{
					{RackID: "1", Tags: []string{"tag2"}},
					{RackID: "2", Tags: []string{"tag1"}},
					{RackID: "3", Tags: []string{"tag1", "tag2"}},
					{RackID: "2", Tags: []string{"tag1"}},
				},
				tags: []string{"tag1", "tag2"},
			},
			want: metal.Machine{RackID: "1"},
		},
		{
			name: "preferred racks aren't available",
			args: args{
				allMachines: metal.Machines{
					{RackID: "3"},
				},
				projectMachines: metal.Machines{
					{RackID: "1", Tags: []string{"tag1"}},
					{RackID: "2", Tags: []string{"tag1"}},
					{RackID: "3", Tags: []string{"tag1", "tag2"}},
					{RackID: "3", Tags: []string{"tag2"}},
					{RackID: "2", Tags: []string{"tag2"}},
				},
				tags: []string{"tag1", "tag2"},
			},
			want: metal.Machine{RackID: "3"},
		},
		{
			name: "racks preferred by tags aren't available, choose by project",
			args: args{
				allMachines: metal.Machines{
					{RackID: "3"},
					{RackID: "2"},
					{RackID: "2"},
					{RackID: "2"},
					{RackID: "2"},
					{RackID: "2"},
					{RackID: "2"},
				},
				projectMachines: metal.Machines{
					{RackID: "1", Tags: []string{"tag1"}},
					{RackID: "2", Tags: []string{"tag1"}},
					{RackID: "2", Tags: []string{"tag1"}},
					{RackID: "3", Tags: []string{"tag1"}},
					{RackID: "3", Tags: []string{"tag1"}},
					{RackID: "1"},
					{RackID: "1"},
					{RackID: "2"},
				},
				tags: []string{"tag1"},
			},
			want: metal.Machine{RackID: "3"},
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

func Test_intersect(t *testing.T) {
	type args struct {
		a []string
		b []string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "both empty",
			want: []string{},
		},
		{
			name: "one empty",
			args: args{
				a: []string{""},
			},
			want: []string{},
		},
		{
			name: "empty intersection",
			args: args{
				a: []string{"1"},
				b: []string{"2"},
			},
			want: []string{},
		},
		{
			name: "non-empty intersection",
			args: args{
				a: []string{"1", "2", "3"},
				b: []string{"1", "3", "4"},
			},
			want: []string{"1", "3"},
		},
		{
			name: "intersection equals a",
			args: args{
				a: []string{"3", "2", "1"},
				b: []string{"1", "3", "4", "2"},
			},
			want: []string{"3", "2", "1"},
		},
		{
			name: "intersection contains same elements as b",
			args: args{
				a: []string{"3", "2", "1", "4"},
				b: []string{"1", "3", "4"},
			},
			want: []string{"3", "1", "4"},
		},
		{
			name: "a and b contain same elements",
			args: args{
				a: []string{"3", "2", "1", "4"},
				b: []string{"1", "3", "4", "2"},
			},
			want: []string{"3", "2", "1", "4"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := intersect(tt.args.a, tt.args.b); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("intersect() = %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkElectMachine(b *testing.B) {
	type args struct {
		allMachines     metal.Machines
		projectMachines metal.Machines
		tags            []string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "10 available, 11 project",
			args: args{
				allMachines:     getTestMachines(2, []string{"1", "2", "3", "4", "5"}, []string{}),
				projectMachines: append(getTestMachines(2, []string{"1", "2", "3", "4", "5"}, []string{"tag1", "tag2", "tag3", "tag4"}), getTestMachines(1, []string{"6"}, []string{})...),
				tags:            []string{"tag1", "tag2", "tag3", "tag4"},
			},
		},
		{
			name: "100 available",
			args: args{
				allMachines:     getTestMachines(20, []string{"1", "2", "3", "4", "5"}, []string{}),
				projectMachines: append(getTestMachines(20, []string{"1", "2", "3", "4", "5"}, []string{"tag1", "tag2", "tag3", "tag4"}), getTestMachines(10, []string{"6"}, []string{})...),
				tags:            []string{"tag1", "tag2", "tag3", "tag4"},
			},
		},
		{
			name: "1000 available, 1100 project",
			args: args{
				allMachines:     getTestMachines(200, []string{"1", "2", "3", "4", "5"}, []string{}),
				projectMachines: append(getTestMachines(200, []string{"1", "2", "3", "4", "5"}, []string{"tag1", "tag2", "tag3", "tag4"}), getTestMachines(100, []string{"6"}, []string{})...),
				tags:            []string{"tag1", "tag2", "tag3", "tag4"},
			},
		},
	}
	for _, t := range tests {
		b.Run(t.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				electMachine(t.args.allMachines, t.args.projectMachines, t.args.tags)
			}
		})
	}
}

func getTestMachines(numPerRack int, rackids []string, tags []string) metal.Machines {
	machines := make(metal.Machines, 0)

	for _, id := range rackids {
		for i := 0; i < numPerRack; i++ {
			m := metal.Machine{
				RackID: id,
				Tags:   tags,
			}

			machines = append(machines, m)
		}
	}

	return machines
}
