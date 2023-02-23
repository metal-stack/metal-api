//go:build integration
// +build integration

package datastore

import (
	"testing"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"github.com/metal-stack/metal-lib/pkg/pointer"
)

type machineTestable struct{}

func (_ *machineTestable) wipe() error {
	_, err := sharedDS.machineTable().Delete().RunWrite(sharedDS.session)
	return err
}

func (_ *machineTestable) create(s *metal.Machine) error {
	return sharedDS.CreateMachine(s)
}

func (_ *machineTestable) delete(id string) error {
	return sharedDS.DeleteMachine(&metal.Machine{Base: metal.Base{ID: id}})
}

func (_ *machineTestable) update(old *metal.Machine, mutateFn func(s *metal.Machine)) error {
	mod := *old
	if mutateFn != nil {
		mutateFn(&mod)
	}

	return sharedDS.UpdateMachine(old, &mod)
}

func (_ *machineTestable) find(id string) (*metal.Machine, error) {
	return sharedDS.FindMachineByID(id)
}

func (_ *machineTestable) list() ([]*metal.Machine, error) {
	res, err := sharedDS.ListMachines()
	if err != nil {
		return nil, err
	}

	return derefSlice(res), nil
}

func (_ *machineTestable) search(q *MachineSearchQuery) ([]*metal.Machine, error) {
	var res metal.Machines
	err := sharedDS.SearchMachines(q, &res)
	if err != nil {
		return nil, err
	}

	return derefSlice(res), nil
}

func (_ *machineTestable) defaultBody(m *metal.Machine) *metal.Machine {
	m.Hardware = metal.MachineHardware{Nics: metal.Nics{}, Disks: []metal.BlockDevice{}}
	m.Tags = []string{}
	return m
}

func TestRethinkStore_FindMachine(t *testing.T) {
	tt := &machineTestable{}
	defer func() {
		tt.wipe()
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
		tt.wipe()
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
	}

	for i := range tests {
		tests[i].run(t, tt)
	}
}

func TestRethinkStore_ListMachines(t *testing.T) {
	tt := &machineTestable{}
	defer func() {
		tt.wipe()
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
		tt.wipe()
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
		tt.wipe()
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
		tt.wipe()
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
				Hardware: metal.MachineHardware{Nics: metal.Nics{}, Disks: []metal.BlockDevice{}},
				Tags:     []string{"a=b"},
			},
		},
	}
	for i := range tests {
		tests[i].run(t, tt)
	}
}
