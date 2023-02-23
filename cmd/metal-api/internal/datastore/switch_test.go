//go:build integration
// +build integration

package datastore

import (
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"github.com/metal-stack/metal-lib/pkg/pointer"
	"github.com/metal-stack/metal-lib/pkg/testcommon"
	"github.com/stretchr/testify/require"
)

func TestRethinkStore_FindSwitch(t *testing.T) {
	defer func() {
		wipeSwitchTable(t, sharedDS)
	}()

	tests := []struct {
		name    string
		id      string
		mock    metal.Switches
		want    *metal.Switch
		wantErr error
	}{
		{
			name: "find",
			id:   "2",
			mock: metal.Switches{
				{Base: metal.Base{ID: "1"}},
				{Base: metal.Base{ID: "2"}},
				{Base: metal.Base{ID: "3"}},
			},
			want: &metal.Switch{
				Base: metal.Base{ID: "2"}, Nics: metal.Nics{},
			},
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
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			wipeSwitchTable(t, sharedDS)

			for _, s := range tt.mock {
				s := s
				err := sharedDS.CreateSwitch(&s)
				require.NoError(t, err)
			}

			got, err := sharedDS.FindSwitch(tt.id)

			if diff := cmp.Diff(tt.wantErr, err, testcommon.ErrorStringComparer()); diff != "" {
				t.Errorf("error diff (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tt.want, got, ignoreTimestamps()); diff != "" {
				t.Errorf("diff (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRethinkStore_SearchSwitches(t *testing.T) {
	defer func() {
		wipeSwitchTable(t, sharedDS)
	}()

	tests := []struct {
		name    string
		q       *SwitchSearchQuery
		mock    metal.Switches
		want    metal.Switches
		wantErr error
	}{
		{
			name: "empty result",
			q: &SwitchSearchQuery{
				ID: pointer.Pointer("2"),
			},
			mock: metal.Switches{
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
			mock: metal.Switches{
				{Base: metal.Base{ID: "1"}},
				{Base: metal.Base{ID: "2"}},
				{Base: metal.Base{ID: "3"}},
			},
			want: metal.Switches{
				{Base: metal.Base{ID: "2"}, Nics: metal.Nics{}},
			},
			wantErr: nil,
		},
		{
			name: "search by partition",
			q: &SwitchSearchQuery{
				PartitionID: pointer.Pointer("b"),
			},
			mock: metal.Switches{
				{Base: metal.Base{ID: "1"}, PartitionID: "a"},
				{Base: metal.Base{ID: "2"}, PartitionID: "b"},
				{Base: metal.Base{ID: "3"}, PartitionID: "c"},
				{Base: metal.Base{ID: "4"}, PartitionID: "b"},
			},
			want: metal.Switches{
				{Base: metal.Base{ID: "2"}, PartitionID: "b", Nics: metal.Nics{}},
				{Base: metal.Base{ID: "4"}, PartitionID: "b", Nics: metal.Nics{}},
			},
			wantErr: nil,
		},
		{
			name: "search by rack",
			q: &SwitchSearchQuery{
				RackID: pointer.Pointer("b"),
			},
			mock: metal.Switches{
				{Base: metal.Base{ID: "1"}, RackID: "a"},
				{Base: metal.Base{ID: "2"}, RackID: "b"},
				{Base: metal.Base{ID: "3"}, RackID: "c"},
			},
			want: metal.Switches{
				{Base: metal.Base{ID: "2"}, RackID: "b", Nics: metal.Nics{}},
			},
			wantErr: nil,
		},
		{
			name: "search by os vendor",
			q: &SwitchSearchQuery{
				OSVendor: pointer.Pointer("sonic"),
			},
			mock: metal.Switches{
				{Base: metal.Base{ID: "1"}, OS: &metal.SwitchOS{Vendor: "cumulus"}},
				{Base: metal.Base{ID: "2"}, OS: &metal.SwitchOS{Vendor: "sonic"}},
				{Base: metal.Base{ID: "3"}, OS: &metal.SwitchOS{Vendor: "sonic"}},
			},
			want: metal.Switches{
				{Base: metal.Base{ID: "2"}, OS: &metal.SwitchOS{Vendor: "sonic"}, Nics: metal.Nics{}},
				{Base: metal.Base{ID: "3"}, OS: &metal.SwitchOS{Vendor: "sonic"}, Nics: metal.Nics{}},
			},
			wantErr: nil,
		},
		{
			name: "search by os version",
			q: &SwitchSearchQuery{
				OSVersion: pointer.Pointer("1.2.3"),
			},
			mock: metal.Switches{
				{Base: metal.Base{ID: "1"}, OS: &metal.SwitchOS{Version: "1.2.1"}},
				{Base: metal.Base{ID: "2"}, OS: &metal.SwitchOS{Version: "1.2.2"}},
				{Base: metal.Base{ID: "3"}, OS: &metal.SwitchOS{Version: "1.2.3"}},
			},
			want: metal.Switches{
				{Base: metal.Base{ID: "3"}, OS: &metal.SwitchOS{Version: "1.2.3"}, Nics: metal.Nics{}},
			},
			wantErr: nil,
		},
	}

	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			wipeSwitchTable(t, sharedDS)

			for _, s := range tt.mock {
				s := s
				err := sharedDS.CreateSwitch(&s)
				require.NoError(t, err)
			}

			var got metal.Switches
			err := sharedDS.SearchSwitches(tt.q, &got)
			sort.Slice(got, func(i, j int) bool { return got[i].ID < got[j].ID })

			if diff := cmp.Diff(tt.wantErr, err, testcommon.ErrorStringComparer()); diff != "" {
				t.Errorf("error diff (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tt.want, got, ignoreTimestamps()); diff != "" {
				t.Errorf("diff (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRethinkStore_ListSwitches(t *testing.T) {
	defer func() {
		wipeSwitchTable(t, sharedDS)
	}()

	tests := []struct {
		name    string
		mock    metal.Switches
		want    metal.Switches
		wantErr error
	}{
		{
			name: "list",
			mock: metal.Switches{
				{Base: metal.Base{ID: "1"}},
				{Base: metal.Base{ID: "2"}},
				{Base: metal.Base{ID: "3"}},
			},
			want: metal.Switches{
				{Base: metal.Base{ID: "1"}, Nics: metal.Nics{}},
				{Base: metal.Base{ID: "2"}, Nics: metal.Nics{}},
				{Base: metal.Base{ID: "3"}, Nics: metal.Nics{}},
			},
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			wipeSwitchTable(t, sharedDS)

			for _, s := range tt.mock {
				s := s
				err := sharedDS.CreateSwitch(&s)
				require.NoError(t, err)
			}

			got, err := sharedDS.ListSwitches()
			sort.Slice(got, func(i, j int) bool { return got[i].ID < got[j].ID })

			if diff := cmp.Diff(tt.wantErr, err, testcommon.ErrorStringComparer()); diff != "" {
				t.Errorf("error diff (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tt.want, got, ignoreTimestamps()); diff != "" {
				t.Errorf("diff (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRethinkStore_CreateSwitch(t *testing.T) {
	defer func() {
		wipeSwitchTable(t, sharedDS)
	}()

	tests := []struct {
		name    string
		mock    metal.Switches
		want    *metal.Switch
		wantErr error
	}{
		{
			name: "create",
			want: &metal.Switch{
				Base: metal.Base{ID: "1"}, Nics: metal.Nics{},
			},
			wantErr: nil,
		},
		{
			name: "already exists",
			mock: metal.Switches{
				{Base: metal.Base{ID: "1"}},
			},
			want: &metal.Switch{
				Base: metal.Base{ID: "1"}, Nics: metal.Nics{},
			},
			wantErr: metal.Conflict(`cannot create switch in database, entity already exists: 1`),
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			wipeSwitchTable(t, sharedDS)

			for _, s := range tt.mock {
				s := s
				err := sharedDS.CreateSwitch(&s)
				require.NoError(t, err)
			}

			err := sharedDS.CreateSwitch(tt.want)
			if diff := cmp.Diff(tt.wantErr, err, testcommon.ErrorStringComparer()); diff != "" {
				t.Errorf("error diff (-want +got):\n%s", diff)
			}

			if tt.wantErr == nil {
				got, err := sharedDS.FindSwitch(tt.want.ID)
				require.NoError(t, err)

				if diff := cmp.Diff(tt.want, got, ignoreTimestamps()); diff != "" {
					t.Errorf("diff (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestRethinkStore_DeleteSwitch(t *testing.T) {
	defer func() {
		wipeSwitchTable(t, sharedDS)
	}()

	tests := []struct {
		name    string
		id      string
		mock    metal.Switches
		want    metal.Switches
		wantErr error
	}{
		{
			name: "delete",
			id:   "2",
			mock: metal.Switches{
				{Base: metal.Base{ID: "1"}},
				{Base: metal.Base{ID: "2"}},
				{Base: metal.Base{ID: "3"}},
			},
			want: metal.Switches{
				{Base: metal.Base{ID: "1"}, Nics: metal.Nics{}},
				{Base: metal.Base{ID: "3"}, Nics: metal.Nics{}},
			},
		},
		{
			name: "not exists results in noop",
			id:   "abc",
			mock: metal.Switches{
				{Base: metal.Base{ID: "1"}},
				{Base: metal.Base{ID: "2"}},
				{Base: metal.Base{ID: "3"}},
			},
			want: metal.Switches{
				{Base: metal.Base{ID: "1"}, Nics: metal.Nics{}},
				{Base: metal.Base{ID: "2"}, Nics: metal.Nics{}},
				{Base: metal.Base{ID: "3"}, Nics: metal.Nics{}},
			},
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			wipeSwitchTable(t, sharedDS)

			for _, s := range tt.mock {
				s := s
				err := sharedDS.CreateSwitch(&s)
				require.NoError(t, err)
			}

			err := sharedDS.DeleteSwitch(&metal.Switch{Base: metal.Base{ID: tt.id}})
			if diff := cmp.Diff(tt.wantErr, err, testcommon.ErrorStringComparer()); diff != "" {
				t.Errorf("error diff (-want +got):\n%s", diff)
			}

			if tt.wantErr == nil {
				got, err := sharedDS.ListSwitches()
				require.NoError(t, err)

				sort.Slice(got, func(i, j int) bool { return got[i].ID < got[j].ID })

				if diff := cmp.Diff(tt.want, got, ignoreTimestamps()); diff != "" {
					t.Errorf("diff (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestRethinkStore_UpdateSwitch(t *testing.T) {
	defer func() {
		wipeSwitchTable(t, sharedDS)
	}()

	tests := []struct {
		name     string
		mock     metal.Switches
		mutateFn func(s *metal.Switch)
		want     *metal.Switch
		wantErr  error
	}{
		{
			name: "update",
			mock: metal.Switches{
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
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			wipeSwitchTable(t, sharedDS)

			for _, s := range tt.mock {
				s := s
				err := sharedDS.CreateSwitch(&s)
				require.NoError(t, err)
			}

			old, err := sharedDS.FindSwitch(tt.want.ID)
			require.NoError(t, err)

			mod := *old
			if tt.mutateFn != nil {
				tt.mutateFn(&mod)
			}

			err = sharedDS.UpdateSwitch(old, &mod)
			if diff := cmp.Diff(tt.wantErr, err, testcommon.ErrorStringComparer()); diff != "" {
				t.Errorf("error diff (-want +got):\n%s", diff)
			}

			got, err := sharedDS.FindSwitch(tt.want.ID)
			require.NoError(t, err)

			if diff := cmp.Diff(tt.want, got, ignoreTimestamps()); diff != "" {
				t.Errorf("diff (-want +got):\n%s", diff)
			}
		})
	}
}

func wipeSwitchTable(t *testing.T, ds *RethinkStore) {
	_, err := ds.switchTable().Delete().RunWrite(ds.session)
	require.NoError(t, err)
}
