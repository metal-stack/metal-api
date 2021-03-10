package datastore

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/testdata"
)

func TestRethinkStore_FindNetworkByID(t *testing.T) {
	// mock the DB
	ds, mock := InitMockDB()
	testdata.InitMockDBData(mock)

	type args struct {
		id string
	}
	tests := []struct {
		name    string
		rs      *RethinkStore
		args    args
		want    *metal.Network
		wantErr bool
	}{
		// Test Data Array:
		{
			name: "TestRethinkStore_FindNetworkByID Test 1",
			rs:   ds,
			args: args{
				id: "1",
			},
			want:    &testdata.Nw1,
			wantErr: false,
		},
		{
			name: "TestRethinkStore_FindNetworkByID Test 2",
			rs:   ds,
			args: args{
				id: "2",
			},
			want:    &testdata.Nw2,
			wantErr: false,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.rs.FindNetworkByID(tt.args.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.FindNetworkByID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("RethinkStore.FindNetworkByID() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRethinkStore_ListNetworks(t *testing.T) {
	// mock the DBs
	ds, mock := InitMockDB()
	testdata.InitMockDBData(mock)

	tests := []struct {
		name    string
		rs      *RethinkStore
		want    metal.Networks
		wantErr bool
	}{
		// Test-Data List / Test Cases:
		{
			name:    "TestRethinkStore_ListNetworks Test 1",
			rs:      ds,
			want:    testdata.TestNetworks,
			wantErr: false,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.rs.ListNetworks()
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.ListNetworks() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("RethinkStore.ListNetworks() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
