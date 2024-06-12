//go:build integration
// +build integration

package datastore

import (
	"testing"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"github.com/metal-stack/metal-lib/pkg/pointer"
	"github.com/stretchr/testify/require"
)

type sizeTestable struct{}

func (_ *sizeTestable) wipe() error {
	_, err := sharedDS.sizeTable().Delete().RunWrite(sharedDS.session)
	return err
}

func (_ *sizeTestable) create(s *metal.Size) error { // nolint:unused
	return sharedDS.CreateSize(s)
}

func (_ *sizeTestable) delete(id string) error { // nolint:unused
	return sharedDS.DeleteSize(&metal.Size{Base: metal.Base{ID: id}})
}

func (_ *sizeTestable) update(old *metal.Size, mutateFn func(s *metal.Size)) error { // nolint:unused
	mod := *old
	if mutateFn != nil {
		mutateFn(&mod)
	}

	return sharedDS.UpdateSize(old, &mod)
}

func (_ *sizeTestable) find(id string) (*metal.Size, error) { // nolint:unused
	return sharedDS.FindSize(id)
}

func (_ *sizeTestable) list() ([]*metal.Size, error) { // nolint:unused
	res, err := sharedDS.ListSizes()
	if err != nil {
		return nil, err
	}

	return derefSlice(res), nil
}

func (_ *sizeTestable) search(q *SizeSearchQuery) ([]*metal.Size, error) { // nolint:unused
	var res metal.Sizes
	err := sharedDS.SearchSizes(q, &res)
	if err != nil {
		return nil, err
	}

	return derefSlice(res), nil
}

func (_ *sizeTestable) defaultBody(s *metal.Size) *metal.Size {
	if s.Constraints == nil {
		s.Constraints = []metal.Constraint{}
	}
	if s.Reservations == nil {
		s.Reservations = metal.Reservations{}
	}
	for i := range s.Reservations {
		if s.Reservations[i].PartitionIDs == nil {
			s.Reservations[i].PartitionIDs = []string{}
		}
	}
	return s
}

func TestRethinkStore_FindSize(t *testing.T) {
	tt := &sizeTestable{}
	defer func() {
		require.NoError(t, tt.wipe())
	}()

	tests := []findTest[*metal.Size, *SizeSearchQuery]{
		{
			name: "find",
			id:   "2",

			mock: []*metal.Size{
				{Base: metal.Base{ID: "1"}},
				{Base: metal.Base{ID: "2"}},
				{Base: metal.Base{ID: "3"}},
			},
			want:    tt.defaultBody(&metal.Size{Base: metal.Base{ID: "2"}}),
			wantErr: nil,
		},
		{
			name:    "not found",
			id:      "4",
			want:    nil,
			wantErr: metal.NotFound(`no size with id "4" found`),
		},
	}
	for i := range tests {
		tests[i].run(t, tt)
	}
}

func TestRethinkStore_SearchSizes(t *testing.T) {
	tt := &sizeTestable{}
	defer func() {
		require.NoError(t, tt.wipe())
	}()

	tests := []searchTest[*metal.Size, *SizeSearchQuery]{
		{
			name: "empty result",
			q: &SizeSearchQuery{
				ID: pointer.Pointer("2"),
			},
			mock: []*metal.Size{
				{Base: metal.Base{ID: "1"}},
			},
			want:    nil,
			wantErr: nil,
		},
		{
			name: "search by id",
			q: &SizeSearchQuery{
				ID: pointer.Pointer("2"),
			},
			mock: []*metal.Size{
				{Base: metal.Base{ID: "1"}},
				{Base: metal.Base{ID: "2"}},
				{Base: metal.Base{ID: "3"}},
			},
			want: []*metal.Size{
				tt.defaultBody(&metal.Size{Base: metal.Base{ID: "2"}}),
			},
			wantErr: nil,
		},
		{
			name: "search by name",
			q: &SizeSearchQuery{
				Name: pointer.Pointer("b"),
			},
			mock: []*metal.Size{
				{Base: metal.Base{ID: "1", Name: "a"}},
				{Base: metal.Base{ID: "2", Name: "b"}},
				{Base: metal.Base{ID: "3", Name: "c"}},
			},
			want: []*metal.Size{
				tt.defaultBody(&metal.Size{Base: metal.Base{ID: "2", Name: "b"}}),
			},
			wantErr: nil,
		},
		{
			name: "search reservation project",
			q: &SizeSearchQuery{
				ReservationsProject: pointer.Pointer("2"),
			},
			mock: []*metal.Size{
				{Base: metal.Base{ID: "1"}, Reservations: metal.Reservations{{ProjectID: "1"}}},
				{Base: metal.Base{ID: "2"}, Reservations: metal.Reservations{{ProjectID: "2"}}},
				{Base: metal.Base{ID: "3"}, Reservations: metal.Reservations{{ProjectID: "3"}}},
			},
			want: []*metal.Size{
				tt.defaultBody(&metal.Size{Base: metal.Base{ID: "2"}, Reservations: metal.Reservations{{ProjectID: "2"}}}),
			},
			wantErr: nil,
		},
		{
			name: "search reservation partition",
			q: &SizeSearchQuery{
				ReservationsPartition: pointer.Pointer("p1"),
			},
			mock: []*metal.Size{
				{Base: metal.Base{ID: "1"}, Reservations: metal.Reservations{{PartitionIDs: []string{"p1"}}}},
				{Base: metal.Base{ID: "2"}, Reservations: metal.Reservations{{PartitionIDs: []string{"p1", "p2"}}}},
				{Base: metal.Base{ID: "3"}, Reservations: metal.Reservations{{PartitionIDs: []string{"p3"}}}},
			},
			want: []*metal.Size{
				tt.defaultBody(&metal.Size{Base: metal.Base{ID: "1"}, Reservations: metal.Reservations{{PartitionIDs: []string{"p1"}}}}),
				tt.defaultBody(&metal.Size{Base: metal.Base{ID: "2"}, Reservations: metal.Reservations{{PartitionIDs: []string{"p1", "p2"}}}}),
			},
			wantErr: nil,
		},
	}

	for i := range tests {
		tests[i].run(t, tt)
	}
}

func TestRethinkStore_ListSizes(t *testing.T) {
	tt := &sizeTestable{}
	defer func() {
		require.NoError(t, tt.wipe())
	}()

	tests := []listTest[*metal.Size, *SizeSearchQuery]{
		{
			name: "list",
			mock: []*metal.Size{
				{Base: metal.Base{ID: "1"}},
				{Base: metal.Base{ID: "2"}},
				{Base: metal.Base{ID: "3"}},
			},
			want: []*metal.Size{
				tt.defaultBody(&metal.Size{Base: metal.Base{ID: "1"}}),
				tt.defaultBody(&metal.Size{Base: metal.Base{ID: "2"}}),
				tt.defaultBody(&metal.Size{Base: metal.Base{ID: "3"}}),
			},
		},
	}
	for i := range tests {
		tests[i].run(t, tt)
	}
}

func TestRethinkStore_CreateSize(t *testing.T) {
	tt := &sizeTestable{}
	defer func() {
		require.NoError(t, tt.wipe())
	}()

	tests := []createTest[*metal.Size, *SizeSearchQuery]{
		{
			name:    "create",
			want:    tt.defaultBody(&metal.Size{Base: metal.Base{ID: "1"}}),
			wantErr: nil,
		},
		{
			name: "already exists",
			mock: []*metal.Size{
				{Base: metal.Base{ID: "1"}},
			},
			want:    tt.defaultBody(&metal.Size{Base: metal.Base{ID: "1"}}),
			wantErr: metal.Conflict(`cannot create size in database, entity already exists: 1`),
		},
	}
	for i := range tests {
		tests[i].run(t, tt)
	}
}

func TestRethinkStore_DeleteSize(t *testing.T) {
	tt := &sizeTestable{}
	defer func() {
		require.NoError(t, tt.wipe())
	}()

	tests := []deleteTest[*metal.Size, *SizeSearchQuery]{
		{
			name: "delete",
			id:   "2",
			mock: []*metal.Size{
				{Base: metal.Base{ID: "1"}},
				{Base: metal.Base{ID: "2"}},
				{Base: metal.Base{ID: "3"}},
			},
			want: []*metal.Size{
				tt.defaultBody(&metal.Size{Base: metal.Base{ID: "1"}}),
				tt.defaultBody(&metal.Size{Base: metal.Base{ID: "3"}}),
			},
		},
		{
			name: "not exists results in noop",
			id:   "abc",
			mock: []*metal.Size{
				{Base: metal.Base{ID: "1"}},
				{Base: metal.Base{ID: "2"}},
				{Base: metal.Base{ID: "3"}},
			},
			want: []*metal.Size{
				tt.defaultBody(&metal.Size{Base: metal.Base{ID: "1"}}),
				tt.defaultBody(&metal.Size{Base: metal.Base{ID: "2"}}),
				tt.defaultBody(&metal.Size{Base: metal.Base{ID: "3"}}),
			},
		},
	}
	for i := range tests {
		tests[i].run(t, tt)
	}
}

func TestRethinkStore_UpdateSize(t *testing.T) {
	tt := &sizeTestable{}
	defer func() {
		require.NoError(t, tt.wipe())
	}()

	tests := []updateTest[*metal.Size, *SizeSearchQuery]{
		{
			name: "update",
			mock: []*metal.Size{
				{Base: metal.Base{ID: "1"}},
				{Base: metal.Base{ID: "2"}},
				{Base: metal.Base{ID: "3"}},
			},
			mutateFn: func(s *metal.Size) {
				s.Labels = map[string]string{"a": "b"}
			},
			want: tt.defaultBody(&metal.Size{
				Base:   metal.Base{ID: "1"},
				Labels: map[string]string{"a": "b"},
			}),
		},
	}
	for i := range tests {
		tests[i].run(t, tt)
	}
}
