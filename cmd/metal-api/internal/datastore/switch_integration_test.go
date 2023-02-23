//go:build integration
// +build integration

package datastore

import (
	"testing"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"github.com/metal-stack/metal-lib/pkg/pointer"
)

type switchTestable struct{}

func (_ *switchTestable) wipe() error {
	_, err := sharedDS.switchTable().Delete().RunWrite(sharedDS.session)
	return err
}

func (_ *switchTestable) create(s *metal.Switch) error {
	return sharedDS.CreateSwitch(s)
}

func (_ *switchTestable) delete(id string) error {
	return sharedDS.DeleteSwitch(&metal.Switch{Base: metal.Base{ID: id}})
}

func (_ *switchTestable) update(old *metal.Switch, mutateFn func(s *metal.Switch)) error {
	mod := *old
	if mutateFn != nil {
		mutateFn(&mod)
	}

	return sharedDS.UpdateSwitch(old, &mod)
}

func (_ *switchTestable) find(id string) (*metal.Switch, error) {
	return sharedDS.FindSwitch(id)
}

func (_ *switchTestable) list() ([]*metal.Switch, error) {
	res, err := sharedDS.ListSwitches()
	if err != nil {
		return nil, err
	}

	return derefSlice(res), nil
}

func (_ *switchTestable) search(q *SwitchSearchQuery) ([]*metal.Switch, error) {
	var res metal.Switches
	err := sharedDS.SearchSwitches(q, &res)
	if err != nil {
		return nil, err
	}

	return derefSlice(res), nil
}

func (_ *switchTestable) defaultBody(m *metal.Switch) *metal.Switch {
	m.Nics = metal.Nics{}
	return m
}

func TestRethinkStore_FindSwitch(t *testing.T) {
	tt := &switchTestable{}
	defer func() {
		tt.wipe()
	}()

	tests := []findTest[*metal.Switch, *SwitchSearchQuery]{
		{
			name: "find",
			id:   "2",

			mock: []*metal.Switch{
				{Base: metal.Base{ID: "1"}},
				{Base: metal.Base{ID: "2"}},
				{Base: metal.Base{ID: "3"}},
			},
			want:    tt.defaultBody(&metal.Switch{Base: metal.Base{ID: "2"}}),
			wantErr: nil,
		},
		{
			name:    "not found",
			id:      "4",
			want:    nil,
			wantErr: metal.NotFound(`no switch with id "4" found`),
		},
	}
	for i := range tests {
		tests[i].run(t, tt)
	}
}

func TestRethinkStore_SearchSwitches(t *testing.T) {
	tt := &switchTestable{}
	defer func() {
		tt.wipe()
	}()

	tests := []searchTest[*metal.Switch, *SwitchSearchQuery]{
		{
			name: "empty result",
			q: &SwitchSearchQuery{
				ID: pointer.Pointer("2"),
			},
			mock: []*metal.Switch{
				{Base: metal.Base{ID: "1"}},
			},
			want:    nil,
			wantErr: nil,
		},
		{
			name: "search by id",
			q: &SwitchSearchQuery{
				ID: pointer.Pointer("2"),
			},
			mock: []*metal.Switch{
				{Base: metal.Base{ID: "1"}},
				{Base: metal.Base{ID: "2"}},
				{Base: metal.Base{ID: "3"}},
			},
			want: []*metal.Switch{
				tt.defaultBody(&metal.Switch{Base: metal.Base{ID: "2"}}),
			},
			wantErr: nil,
		},
		{
			name: "search by partition",
			q: &SwitchSearchQuery{
				PartitionID: pointer.Pointer("b"),
			},
			mock: []*metal.Switch{
				{Base: metal.Base{ID: "1"}, PartitionID: "a"},
				{Base: metal.Base{ID: "2"}, PartitionID: "b"},
				{Base: metal.Base{ID: "3"}, PartitionID: "c"},
				{Base: metal.Base{ID: "4"}, PartitionID: "b"},
			},
			want: []*metal.Switch{
				tt.defaultBody(&metal.Switch{Base: metal.Base{ID: "2"}, PartitionID: "b"}),
				tt.defaultBody(&metal.Switch{Base: metal.Base{ID: "4"}, PartitionID: "b"}),
			},
			wantErr: nil,
		},
		{
			name: "search by rack",
			q: &SwitchSearchQuery{
				RackID: pointer.Pointer("b"),
			},
			mock: []*metal.Switch{
				{Base: metal.Base{ID: "1"}, RackID: "a"},
				{Base: metal.Base{ID: "2"}, RackID: "b"},
				{Base: metal.Base{ID: "3"}, RackID: "c"},
			},
			want: []*metal.Switch{
				tt.defaultBody(&metal.Switch{Base: metal.Base{ID: "2"}, RackID: "b"}),
			},
			wantErr: nil,
		},
		{
			name: "search by os vendor",
			q: &SwitchSearchQuery{
				OSVendor: pointer.Pointer("sonic"),
			},
			mock: []*metal.Switch{
				{Base: metal.Base{ID: "1"}, OS: &metal.SwitchOS{Vendor: "cumulus"}},
				{Base: metal.Base{ID: "2"}, OS: &metal.SwitchOS{Vendor: "sonic"}},
				{Base: metal.Base{ID: "3"}, OS: &metal.SwitchOS{Vendor: "sonic"}},
			},
			want: []*metal.Switch{
				tt.defaultBody(&metal.Switch{Base: metal.Base{ID: "2"}, OS: &metal.SwitchOS{Vendor: "sonic"}}),
				tt.defaultBody(&metal.Switch{Base: metal.Base{ID: "3"}, OS: &metal.SwitchOS{Vendor: "sonic"}}),
			},
			wantErr: nil,
		},
		{
			name: "search by os version",
			q: &SwitchSearchQuery{
				OSVersion: pointer.Pointer("1.2.3"),
			},
			mock: []*metal.Switch{
				{Base: metal.Base{ID: "1"}, OS: &metal.SwitchOS{Version: "1.2.1"}},
				{Base: metal.Base{ID: "2"}, OS: &metal.SwitchOS{Version: "1.2.2"}},
				{Base: metal.Base{ID: "3"}, OS: &metal.SwitchOS{Version: "1.2.3"}},
			},
			want: []*metal.Switch{
				tt.defaultBody(&metal.Switch{Base: metal.Base{ID: "3"}, OS: &metal.SwitchOS{Version: "1.2.3"}}),
			},
			wantErr: nil,
		},
	}

	for i := range tests {
		tests[i].run(t, tt)
	}
}

func TestRethinkStore_ListSwitches(t *testing.T) {
	tt := &switchTestable{}
	defer func() {
		tt.wipe()
	}()

	tests := []listTest[*metal.Switch, *SwitchSearchQuery]{
		{
			name: "list",
			mock: []*metal.Switch{
				{Base: metal.Base{ID: "1"}},
				{Base: metal.Base{ID: "2"}},
				{Base: metal.Base{ID: "3"}},
			},
			want: []*metal.Switch{
				tt.defaultBody(&metal.Switch{Base: metal.Base{ID: "1"}}),
				tt.defaultBody(&metal.Switch{Base: metal.Base{ID: "2"}}),
				tt.defaultBody(&metal.Switch{Base: metal.Base{ID: "3"}}),
			},
		},
	}
	for i := range tests {
		tests[i].run(t, tt)
	}
}

func TestRethinkStore_CreateSwitch(t *testing.T) {
	tt := &switchTestable{}
	defer func() {
		tt.wipe()
	}()

	tests := []createTest[*metal.Switch, *SwitchSearchQuery]{
		{
			name: "create",
			want: &metal.Switch{
				Base: metal.Base{ID: "1"}, Nics: metal.Nics{},
			},
			wantErr: nil,
		},
		{
			name: "already exists",
			mock: []*metal.Switch{
				{Base: metal.Base{ID: "1"}},
			},
			want:    tt.defaultBody(&metal.Switch{Base: metal.Base{ID: "1"}}),
			wantErr: metal.Conflict(`cannot create switch in database, entity already exists: 1`),
		},
	}
	for i := range tests {
		tests[i].run(t, tt)
	}
}

func TestRethinkStore_DeleteSwitch(t *testing.T) {
	tt := &switchTestable{}
	defer func() {
		tt.wipe()
	}()

	tests := []deleteTest[*metal.Switch, *SwitchSearchQuery]{
		{
			name: "delete",
			id:   "2",
			mock: []*metal.Switch{
				{Base: metal.Base{ID: "1"}},
				{Base: metal.Base{ID: "2"}},
				{Base: metal.Base{ID: "3"}},
			},
			want: []*metal.Switch{
				{Base: metal.Base{ID: "1"}, Nics: metal.Nics{}},
				{Base: metal.Base{ID: "3"}, Nics: metal.Nics{}},
			},
		},
		{
			name: "not exists results in noop",
			id:   "abc",
			mock: []*metal.Switch{
				{Base: metal.Base{ID: "1"}},
				{Base: metal.Base{ID: "2"}},
				{Base: metal.Base{ID: "3"}},
			},
			want: []*metal.Switch{
				tt.defaultBody(&metal.Switch{Base: metal.Base{ID: "1"}}),
				tt.defaultBody(&metal.Switch{Base: metal.Base{ID: "2"}}),
				tt.defaultBody(&metal.Switch{Base: metal.Base{ID: "3"}}),
			},
		},
	}
	for i := range tests {
		tests[i].run(t, tt)
	}
}

func TestRethinkStore_UpdateSwitch(t *testing.T) {
	tt := &switchTestable{}
	defer func() {
		tt.wipe()
	}()

	tests := []updateTest[*metal.Switch, *SwitchSearchQuery]{
		{
			name: "update",
			mock: []*metal.Switch{
				{Base: metal.Base{ID: "1"}},
				{Base: metal.Base{ID: "2"}},
				{Base: metal.Base{ID: "3"}},
			},
			mutateFn: func(s *metal.Switch) {
				s.RackID = "abc"
			},
			want: &metal.Switch{
				Base:   metal.Base{ID: "1"},
				Nics:   metal.Nics{},
				RackID: "abc",
			},
		},
	}
	for i := range tests {
		tests[i].run(t, tt)
	}
}
