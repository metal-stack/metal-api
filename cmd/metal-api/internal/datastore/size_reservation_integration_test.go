//go:build integration
// +build integration

package datastore

import (
	"testing"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"github.com/metal-stack/metal-lib/pkg/pointer"
	"github.com/stretchr/testify/require"
)

type sizeReservationTestable struct{}

func (_ *sizeReservationTestable) wipe() error {
	_, err := sharedDS.sizeReservationTable().Delete().RunWrite(sharedDS.session)
	return err
}

func (_ *sizeReservationTestable) create(s *metal.SizeReservation) error { // nolint:unused
	return sharedDS.CreateSizeReservation(s)
}

func (_ *sizeReservationTestable) delete(id string) error { // nolint:unused
	return sharedDS.DeleteSizeReservation(&metal.SizeReservation{Base: metal.Base{ID: id}})
}

func (_ *sizeReservationTestable) update(old *metal.SizeReservation, mutateFn func(s *metal.SizeReservation)) error { // nolint:unused
	mod := *old
	if mutateFn != nil {
		mutateFn(&mod)
	}

	return sharedDS.UpdateSizeReservation(old, &mod)
}

func (_ *sizeReservationTestable) find(id string) (*metal.SizeReservation, error) { // nolint:unused
	return sharedDS.FindSizeReservation(id)
}

func (_ *sizeReservationTestable) list() ([]*metal.SizeReservation, error) { // nolint:unused
	res, err := sharedDS.ListSizeReservations()
	if err != nil {
		return nil, err
	}

	return derefSlice(res), nil
}

func (_ *sizeReservationTestable) search(q *SizeReservationSearchQuery) ([]*metal.SizeReservation, error) { // nolint:unused
	var res metal.SizeReservations
	err := sharedDS.SearchSizeReservations(q, &res)
	if err != nil {
		return nil, err
	}

	return derefSlice(res), nil
}

func (_ *sizeReservationTestable) defaultBody(s *metal.SizeReservation) *metal.SizeReservation {
	if s.PartitionIDs == nil {
		s.PartitionIDs = []string{}
	}
	return s
}

func TestRethinkStore_FindSizeReservation(t *testing.T) {
	tt := &sizeReservationTestable{}
	defer func() {
		require.NoError(t, tt.wipe())
	}()

	tests := []findTest[*metal.SizeReservation, *SizeReservationSearchQuery]{
		{
			name: "find",
			id:   "2",

			mock: []*metal.SizeReservation{
				{Base: metal.Base{ID: "1"}},
				{Base: metal.Base{ID: "2"}},
				{Base: metal.Base{ID: "3"}},
			},
			want:    tt.defaultBody(&metal.SizeReservation{Base: metal.Base{ID: "2"}}),
			wantErr: nil,
		},
		{
			name:    "not found",
			id:      "4",
			want:    nil,
			wantErr: metal.NotFound(`no sizereservation with id "4" found`),
		},
	}
	for i := range tests {
		tests[i].run(t, tt)
	}
}

func TestRethinkStore_SearchSizeReservations(t *testing.T) {
	tt := &sizeReservationTestable{}
	defer func() {
		require.NoError(t, tt.wipe())
	}()

	tests := []searchTest[*metal.SizeReservation, *SizeReservationSearchQuery]{
		{
			name: "empty result",
			q: &SizeReservationSearchQuery{
				ID: pointer.Pointer("2"),
			},
			mock: []*metal.SizeReservation{
				{Base: metal.Base{ID: "1"}},
			},
			want:    nil,
			wantErr: nil,
		},
		{
			name: "search by id",
			q: &SizeReservationSearchQuery{
				ID: pointer.Pointer("2"),
			},
			mock: []*metal.SizeReservation{
				{Base: metal.Base{ID: "1"}},
				{Base: metal.Base{ID: "2"}},
				{Base: metal.Base{ID: "3"}},
			},
			want: []*metal.SizeReservation{
				tt.defaultBody(&metal.SizeReservation{Base: metal.Base{ID: "2"}}),
			},
			wantErr: nil,
		},
		{
			name: "search by name",
			q: &SizeReservationSearchQuery{
				Name: pointer.Pointer("b"),
			},
			mock: []*metal.SizeReservation{
				{Base: metal.Base{ID: "1", Name: "a"}},
				{Base: metal.Base{ID: "2", Name: "b"}},
				{Base: metal.Base{ID: "3", Name: "c"}},
			},
			want: []*metal.SizeReservation{
				tt.defaultBody(&metal.SizeReservation{Base: metal.Base{ID: "2", Name: "b"}}),
			},
			wantErr: nil,
		},
		{
			name: "search by size",
			q: &SizeReservationSearchQuery{
				SizeID: pointer.Pointer("size-a"),
			},
			mock: []*metal.SizeReservation{
				{Base: metal.Base{ID: "1"}, SizeID: "size-a"},
				{Base: metal.Base{ID: "2"}, SizeID: "size-b"},
				{Base: metal.Base{ID: "3"}, SizeID: "size-c"},
			},
			want: []*metal.SizeReservation{
				tt.defaultBody(&metal.SizeReservation{Base: metal.Base{ID: "1"}, SizeID: "size-a"}),
			},
			wantErr: nil,
		},
		{
			name: "search by label",
			q: &SizeReservationSearchQuery{
				Labels: map[string]string{
					"a": "b",
				},
			},
			mock: []*metal.SizeReservation{
				{Base: metal.Base{ID: "1"}, Labels: map[string]string{"a": "x"}},
				{Base: metal.Base{ID: "2"}, Labels: map[string]string{"a": "b"}},
				{Base: metal.Base{ID: "3"}, Labels: map[string]string{"a": "b"}},
			},
			want: []*metal.SizeReservation{
				tt.defaultBody(&metal.SizeReservation{Base: metal.Base{ID: "2"}, Labels: map[string]string{"a": "b"}}),
				tt.defaultBody(&metal.SizeReservation{Base: metal.Base{ID: "3"}, Labels: map[string]string{"a": "b"}}),
			},
			wantErr: nil,
		},
		{
			name: "search by partition",
			q: &SizeReservationSearchQuery{
				Partition: pointer.Pointer("b"),
			},
			mock: []*metal.SizeReservation{
				{Base: metal.Base{ID: "1"}, PartitionIDs: []string{"b"}},
				{Base: metal.Base{ID: "2"}, PartitionIDs: []string{"a", "b"}},
				{Base: metal.Base{ID: "3"}, PartitionIDs: []string{"a"}},
			},
			want: []*metal.SizeReservation{
				tt.defaultBody(&metal.SizeReservation{Base: metal.Base{ID: "1"}, PartitionIDs: []string{"b"}}),
				tt.defaultBody(&metal.SizeReservation{Base: metal.Base{ID: "2"}, PartitionIDs: []string{"a", "b"}}),
			},
			wantErr: nil,
		},
		{
			name: "search by project",
			q: &SizeReservationSearchQuery{
				Project: pointer.Pointer("3"),
			},
			mock: []*metal.SizeReservation{
				{Base: metal.Base{ID: "1"}, ProjectID: "1"},
				{Base: metal.Base{ID: "2"}, ProjectID: "2"},
				{Base: metal.Base{ID: "3"}, ProjectID: "3"},
			},
			want: []*metal.SizeReservation{
				tt.defaultBody(&metal.SizeReservation{Base: metal.Base{ID: "3"}, ProjectID: "3"}),
			},
			wantErr: nil,
		},
	}

	for i := range tests {
		tests[i].run(t, tt)
	}
}

func TestRethinkStore_ListSizeReservations(t *testing.T) {
	tt := &sizeReservationTestable{}
	defer func() {
		require.NoError(t, tt.wipe())
	}()

	tests := []listTest[*metal.SizeReservation, *SizeReservationSearchQuery]{
		{
			name: "list",
			mock: []*metal.SizeReservation{
				{Base: metal.Base{ID: "1"}},
				{Base: metal.Base{ID: "2"}},
				{Base: metal.Base{ID: "3"}},
			},
			want: []*metal.SizeReservation{
				tt.defaultBody(&metal.SizeReservation{Base: metal.Base{ID: "1"}}),
				tt.defaultBody(&metal.SizeReservation{Base: metal.Base{ID: "2"}}),
				tt.defaultBody(&metal.SizeReservation{Base: metal.Base{ID: "3"}}),
			},
		},
	}
	for i := range tests {
		tests[i].run(t, tt)
	}
}

func TestRethinkStore_CreateSizeReservation(t *testing.T) {
	tt := &sizeReservationTestable{}
	defer func() {
		require.NoError(t, tt.wipe())
	}()

	tests := []createTest[*metal.SizeReservation, *SizeReservationSearchQuery]{
		{
			name:    "create",
			want:    tt.defaultBody(&metal.SizeReservation{Base: metal.Base{ID: "1"}}),
			wantErr: nil,
		},
		{
			name: "already exists",
			mock: []*metal.SizeReservation{
				{Base: metal.Base{ID: "1"}},
			},
			want:    tt.defaultBody(&metal.SizeReservation{Base: metal.Base{ID: "1"}}),
			wantErr: metal.Conflict(`cannot create sizereservation in database, entity already exists: 1`),
		},
	}
	for i := range tests {
		tests[i].run(t, tt)
	}
}

func TestRethinkStore_DeleteSizeReservation(t *testing.T) {
	tt := &sizeReservationTestable{}
	defer func() {
		require.NoError(t, tt.wipe())
	}()

	tests := []deleteTest[*metal.SizeReservation, *SizeReservationSearchQuery]{
		{
			name: "delete",
			id:   "2",
			mock: []*metal.SizeReservation{
				{Base: metal.Base{ID: "1"}},
				{Base: metal.Base{ID: "2"}},
				{Base: metal.Base{ID: "3"}},
			},
			want: []*metal.SizeReservation{
				tt.defaultBody(&metal.SizeReservation{Base: metal.Base{ID: "1"}}),
				tt.defaultBody(&metal.SizeReservation{Base: metal.Base{ID: "3"}}),
			},
		},
		{
			name: "not exists results in noop",
			id:   "abc",
			mock: []*metal.SizeReservation{
				{Base: metal.Base{ID: "1"}},
				{Base: metal.Base{ID: "2"}},
				{Base: metal.Base{ID: "3"}},
			},
			want: []*metal.SizeReservation{
				tt.defaultBody(&metal.SizeReservation{Base: metal.Base{ID: "1"}}),
				tt.defaultBody(&metal.SizeReservation{Base: metal.Base{ID: "2"}}),
				tt.defaultBody(&metal.SizeReservation{Base: metal.Base{ID: "3"}}),
			},
		},
	}
	for i := range tests {
		tests[i].run(t, tt)
	}
}

func TestRethinkStore_UpdateSizeReservation(t *testing.T) {
	tt := &sizeReservationTestable{}
	defer func() {
		require.NoError(t, tt.wipe())
	}()

	tests := []updateTest[*metal.SizeReservation, *SizeReservationSearchQuery]{
		{
			name: "update",
			mock: []*metal.SizeReservation{
				{Base: metal.Base{ID: "1"}},
				{Base: metal.Base{ID: "2"}},
				{Base: metal.Base{ID: "3"}},
			},
			mutateFn: func(s *metal.SizeReservation) {
				s.Labels = map[string]string{"a": "b"}
			},
			want: tt.defaultBody(&metal.SizeReservation{
				Base:   metal.Base{ID: "1"},
				Labels: map[string]string{"a": "b"},
			}),
		},
	}
	for i := range tests {
		tests[i].run(t, tt)
	}
}
