//go:build integration
// +build integration

package datastore

import (
	"testing"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"github.com/metal-stack/metal-lib/pkg/pointer"
	"github.com/stretchr/testify/require"
)

type networkTestable struct{}

func (_ *networkTestable) wipe() error {
	_, err := sharedDS.networkTable().Delete().RunWrite(sharedDS.session)
	return err
}

func (_ *networkTestable) create(m *metal.Network) error { // nolint:unused
	return sharedDS.CreateNetwork(m)
}

func (_ *networkTestable) delete(id string) error { // nolint:unused
	return sharedDS.DeleteNetwork(&metal.Network{Base: metal.Base{ID: id}})
}

func (_ *networkTestable) update(old *metal.Network, mutateFn func(s *metal.Network)) error { // nolint:unused
	mod := *old
	if mutateFn != nil {
		mutateFn(&mod)
	}

	return sharedDS.UpdateNetwork(old, &mod)
}

func (_ *networkTestable) find(id string) (*metal.Network, error) { // nolint:unused
	return sharedDS.FindNetworkByID(id)
}

func (_ *networkTestable) list() ([]*metal.Network, error) { // nolint:unused
	res, err := sharedDS.ListNetworks()
	if err != nil {
		return nil, err
	}

	return derefSlice(res), nil
}

func (_ *networkTestable) search(q *NetworkSearchQuery) ([]*metal.Network, error) { // nolint:unused
	var res metal.Networks
	err := sharedDS.SearchNetworks(q, &res)
	if err != nil {
		return nil, err
	}

	return derefSlice(res), nil
}

func (_ *networkTestable) defaultBody(n *metal.Network) *metal.Network {
	if n.Prefixes == nil {
		n.Prefixes = metal.Prefixes{}
	}
	if n.DestinationPrefixes == nil {
		n.DestinationPrefixes = metal.Prefixes{}
	}
	if n.AdditionalAnnouncableCIDRs == nil {
		n.AdditionalAnnouncableCIDRs = []string{}
	}
	if n.AddressFamilies == nil {
		n.AddressFamilies = metal.AddressFamilies{}
	}
	return n
}

func TestRethinkStore_FindNetwork(t *testing.T) {
	tt := &networkTestable{}
	defer func() {
		require.NoError(t, tt.wipe())
	}()

	tests := []findTest[*metal.Network, *NetworkSearchQuery]{
		{
			name: "find",
			id:   "2",

			mock: []*metal.Network{
				{Base: metal.Base{ID: "1"}},
				{Base: metal.Base{ID: "2"}},
				{Base: metal.Base{ID: "3"}},
			},
			want:    tt.defaultBody(&metal.Network{Base: metal.Base{ID: "2"}}),
			wantErr: nil,
		},
		{
			name:    "not found",
			id:      "4",
			want:    nil,
			wantErr: metal.NotFound(`no network with id "4" found`),
		},
	}
	for i := range tests {
		tests[i].run(t, tt)
	}
}

func TestRethinkStore_SearchNetworks(t *testing.T) {
	tt := &networkTestable{}
	defer func() {
		require.NoError(t, tt.wipe())
	}()

	tests := []searchTest[*metal.Network, *NetworkSearchQuery]{
		{
			name: "empty result",
			q: &NetworkSearchQuery{
				ID: pointer.Pointer("2"),
			},
			mock: []*metal.Network{
				{Base: metal.Base{ID: "1"}},
			},
			want:    nil,
			wantErr: nil,
		},
		{
			name: "search by id",
			q: &NetworkSearchQuery{
				ID: pointer.Pointer("2"),
			},
			mock: []*metal.Network{
				{Base: metal.Base{ID: "1"}},
				{Base: metal.Base{ID: "2"}},
				{Base: metal.Base{ID: "3"}},
			},
			want: []*metal.Network{
				tt.defaultBody(&metal.Network{Base: metal.Base{ID: "2"}}),
			},
			wantErr: nil,
		},
		{
			name: "search by name",
			q: &NetworkSearchQuery{
				Name: pointer.Pointer("b"),
			},
			mock: []*metal.Network{
				{Base: metal.Base{ID: "1", Name: "a"}},
				{Base: metal.Base{ID: "2", Name: "b"}},
				{Base: metal.Base{ID: "3", Name: "c"}},
			},
			want: []*metal.Network{
				tt.defaultBody(&metal.Network{Base: metal.Base{ID: "2", Name: "b"}}),
			},
			wantErr: nil,
		},
		{
			name: "search by partition",
			q: &NetworkSearchQuery{
				PartitionID: pointer.Pointer("b"),
			},
			mock: []*metal.Network{
				{Base: metal.Base{ID: "1"}, PartitionID: "a"},
				{Base: metal.Base{ID: "2"}, PartitionID: "b"},
				{Base: metal.Base{ID: "3"}, PartitionID: "c"},
			},
			want: []*metal.Network{
				tt.defaultBody(&metal.Network{Base: metal.Base{ID: "2"}, PartitionID: "b"}),
			},
			wantErr: nil,
		},
		{
			name: "search by project",
			q: &NetworkSearchQuery{
				ProjectID: pointer.Pointer("b"),
			},
			mock: []*metal.Network{
				{Base: metal.Base{ID: "1"}, ProjectID: "a"},
				{Base: metal.Base{ID: "2"}, ProjectID: "b"},
				{Base: metal.Base{ID: "3"}, ProjectID: "c"},
			},
			want: []*metal.Network{
				tt.defaultBody(&metal.Network{Base: metal.Base{ID: "2"}, ProjectID: "b"}),
			},
			wantErr: nil,
		},
		{
			name: "search by prefix",
			q: &NetworkSearchQuery{
				Prefixes: []string{"1.2.3.4/32", "3.4.5.6/32"},
			},
			mock: []*metal.Network{
				{Base: metal.Base{ID: "1"}, Prefixes: metal.Prefixes{{IP: "1.2.3.4", Length: "32"}}},
				{Base: metal.Base{ID: "2"}, Prefixes: metal.Prefixes{{IP: "1.2.3.4", Length: "32"}, {IP: "3.4.5.6", Length: "32"}}},
				{Base: metal.Base{ID: "3"}, Prefixes: metal.Prefixes{{IP: "255.255.255.0", Length: "24"}}},
			},
			want: []*metal.Network{
				tt.defaultBody(&metal.Network{Base: metal.Base{ID: "2"}, Prefixes: metal.Prefixes{{IP: "1.2.3.4", Length: "32"}, {IP: "3.4.5.6", Length: "32"}}}),
			},
			wantErr: nil,
		},
		{
			name: "search by destination prefix",
			q: &NetworkSearchQuery{
				DestinationPrefixes: []string{"1.2.3.4/32", "3.4.5.6/32"},
			},
			mock: []*metal.Network{
				{Base: metal.Base{ID: "1"}, DestinationPrefixes: metal.Prefixes{{IP: "1.2.3.4", Length: "32"}}},
				{Base: metal.Base{ID: "2"}, DestinationPrefixes: metal.Prefixes{{IP: "1.2.3.4", Length: "32"}, {IP: "3.4.5.6", Length: "32"}}},
				{Base: metal.Base{ID: "3"}, DestinationPrefixes: metal.Prefixes{{IP: "255.255.255.0", Length: "24"}}},
			},
			want: []*metal.Network{
				tt.defaultBody(&metal.Network{Base: metal.Base{ID: "2"}, DestinationPrefixes: metal.Prefixes{{IP: "1.2.3.4", Length: "32"}, {IP: "3.4.5.6", Length: "32"}}}),
			},
			wantErr: nil,
		},
		{
			name: "search by nat",
			q: &NetworkSearchQuery{
				Nat: pointer.Pointer(true),
			},
			mock: []*metal.Network{
				{Base: metal.Base{ID: "1"}, Nat: false},
				{Base: metal.Base{ID: "2"}, Nat: true},
				{Base: metal.Base{ID: "3"}, Nat: false},
			},
			want: []*metal.Network{
				tt.defaultBody(&metal.Network{Base: metal.Base{ID: "2"}, Nat: true}),
			},
			wantErr: nil,
		},
		{
			name: "search by private super",
			q: &NetworkSearchQuery{
				PrivateSuper: pointer.Pointer(true),
			},
			mock: []*metal.Network{
				{Base: metal.Base{ID: "1"}, PrivateSuper: false},
				{Base: metal.Base{ID: "2"}, PrivateSuper: true},
				{Base: metal.Base{ID: "3"}, PrivateSuper: false},
			},
			want: []*metal.Network{
				tt.defaultBody(&metal.Network{Base: metal.Base{ID: "2"}, PrivateSuper: true}),
			},
			wantErr: nil,
		},
		{
			name: "search by underlay",
			q: &NetworkSearchQuery{
				Underlay: pointer.Pointer(false),
			},
			mock: []*metal.Network{
				{Base: metal.Base{ID: "1"}, Underlay: false},
				{Base: metal.Base{ID: "2"}, Underlay: true},
				{Base: metal.Base{ID: "3"}, Underlay: false},
			},
			want: []*metal.Network{
				tt.defaultBody(&metal.Network{Base: metal.Base{ID: "1"}, Underlay: false}),
				tt.defaultBody(&metal.Network{Base: metal.Base{ID: "3"}, Underlay: false}),
			},
			wantErr: nil,
		},
		{
			name: "search by vrf",
			q: &NetworkSearchQuery{
				Vrf: pointer.Pointer(int64(1)),
			},
			mock: []*metal.Network{
				{Base: metal.Base{ID: "1"}, Vrf: 0},
				{Base: metal.Base{ID: "2"}, Vrf: 1},
				{Base: metal.Base{ID: "3"}, Vrf: 2},
			},
			want: []*metal.Network{
				tt.defaultBody(&metal.Network{Base: metal.Base{ID: "2"}, Vrf: 1}),
			},
			wantErr: nil,
		},
		{
			name: "search by parent network id",
			q: &NetworkSearchQuery{
				ParentNetworkID: pointer.Pointer("parent"),
			},
			mock: []*metal.Network{
				{Base: metal.Base{ID: "1"}, ParentNetworkID: "0"},
				{Base: metal.Base{ID: "2"}, ParentNetworkID: "parent"},
				{Base: metal.Base{ID: "3"}, ParentNetworkID: "1"},
			},
			want: []*metal.Network{
				tt.defaultBody(&metal.Network{Base: metal.Base{ID: "2"}, ParentNetworkID: "parent"}),
			},
			wantErr: nil,
		},
		{
			name: "search by labels",
			q: &NetworkSearchQuery{
				Labels: map[string]string{"a": "b"},
			},
			mock: []*metal.Network{
				{Base: metal.Base{ID: "1"}, Labels: nil},
				{Base: metal.Base{ID: "2"}, Labels: map[string]string{"a": "b", "c": "d"}},
				{Base: metal.Base{ID: "3"}, Labels: map[string]string{"c": "d"}},
			},
			want: []*metal.Network{
				tt.defaultBody(&metal.Network{Base: metal.Base{ID: "2"}, Labels: map[string]string{"a": "b", "c": "d"}}),
			},
			wantErr: nil,
		},
	}

	for i := range tests {
		tests[i].run(t, tt)
	}
}

func TestRethinkStore_ListNetworks(t *testing.T) {
	tt := &networkTestable{}
	defer func() {
		require.NoError(t, tt.wipe())
	}()

	tests := []listTest[*metal.Network, *NetworkSearchQuery]{
		{
			name: "list",
			mock: []*metal.Network{
				{Base: metal.Base{ID: "1"}},
				{Base: metal.Base{ID: "2"}},
				{Base: metal.Base{ID: "3"}},
			},
			want: []*metal.Network{
				tt.defaultBody(&metal.Network{Base: metal.Base{ID: "1"}}),
				tt.defaultBody(&metal.Network{Base: metal.Base{ID: "2"}}),
				tt.defaultBody(&metal.Network{Base: metal.Base{ID: "3"}}),
			},
		},
	}
	for i := range tests {
		tests[i].run(t, tt)
	}
}

func TestRethinkStore_CreateNetwork(t *testing.T) {
	tt := &networkTestable{}
	defer func() {
		require.NoError(t, tt.wipe())
	}()

	tests := []createTest[*metal.Network, *NetworkSearchQuery]{
		{
			name:    "create",
			want:    tt.defaultBody(&metal.Network{Base: metal.Base{ID: "1"}}),
			wantErr: nil,
		},
		{
			name: "already exists",
			mock: []*metal.Network{
				{Base: metal.Base{ID: "1"}},
			},
			want:    tt.defaultBody(&metal.Network{Base: metal.Base{ID: "1"}}),
			wantErr: metal.Conflict(`cannot create network in database, entity already exists: 1`),
		},
	}
	for i := range tests {
		tests[i].run(t, tt)
	}
}

func TestRethinkStore_DeleteNetwork(t *testing.T) {
	tt := &networkTestable{}
	defer func() {
		require.NoError(t, tt.wipe())
	}()

	tests := []deleteTest[*metal.Network, *NetworkSearchQuery]{
		{
			name: "delete",
			id:   "2",
			mock: []*metal.Network{
				{Base: metal.Base{ID: "1"}},
				{Base: metal.Base{ID: "2"}},
				{Base: metal.Base{ID: "3"}},
			},
			want: []*metal.Network{
				tt.defaultBody(&metal.Network{Base: metal.Base{ID: "1"}}),
				tt.defaultBody(&metal.Network{Base: metal.Base{ID: "3"}}),
			},
		},
		{
			name: "not exists results in noop",
			id:   "abc",
			mock: []*metal.Network{
				{Base: metal.Base{ID: "1"}},
				{Base: metal.Base{ID: "2"}},
				{Base: metal.Base{ID: "3"}},
			},
			want: []*metal.Network{
				tt.defaultBody(&metal.Network{Base: metal.Base{ID: "1"}}),
				tt.defaultBody(&metal.Network{Base: metal.Base{ID: "2"}}),
				tt.defaultBody(&metal.Network{Base: metal.Base{ID: "3"}}),
			},
		},
	}
	for i := range tests {
		tests[i].run(t, tt)
	}
}

func TestRethinkStore_UpdateNetwork(t *testing.T) {
	tt := &networkTestable{}
	defer func() {
		require.NoError(t, tt.wipe())
	}()

	tests := []updateTest[*metal.Network, *NetworkSearchQuery]{
		{
			name: "update",
			mock: []*metal.Network{
				{Base: metal.Base{ID: "1"}},
				{Base: metal.Base{ID: "2"}},
				{Base: metal.Base{ID: "3"}},
			},
			mutateFn: func(s *metal.Network) {
				s.Labels = map[string]string{"a": "b"}
			},
			want: tt.defaultBody(&metal.Network{
				Base:   metal.Base{ID: "1"},
				Labels: map[string]string{"a": "b"},
			}),
		},
	}
	for i := range tests {
		tests[i].run(t, tt)
	}
}
