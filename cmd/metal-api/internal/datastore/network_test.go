package datastore

import (
	"reflect"
	"testing"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/testdata"
)

func TestRethinkStore_FindPrimaryNetwork(t *testing.T) {

	// mock the DB
	ds, mock := InitMockDB()
	testdata.InitMockDBData(mock)

	tests := []struct {
		name        string
		rs          *RethinkStore
		want        *metal.Network
		wantErr     bool
		partitionID string
	}{
		{
			name:        "TestRethinkStore_GetPrimaryNetwork Test 2",
			rs:          ds,
			want:        &testdata.Nw3,
			partitionID: testdata.Nw3.PartitionID,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := tt.rs.FindPrimaryNetwork(tt.partitionID)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RethinkStore.FindPrimaryNetwork() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRethinkStore_FindNetwork(t *testing.T) {

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
			name: "TestRethinkStore_FindNetwork Test 1",
			rs:   ds,
			args: args{
				id: "1",
			},
			want:    &testdata.Nw1,
			wantErr: false,
		},
		{
			name: "TestRethinkStore_FindNetwork Test 2",
			rs:   ds,
			args: args{
				id: "2",
			},
			want:    &testdata.Nw2,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.rs.FindNetwork(tt.args.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.FindNetwork() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RethinkStore.FindNetwork() = %v, want %v", got, tt.want)
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
		want    []metal.Network
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
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.rs.ListNetworks()
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.ListNetworks() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RethinkStore.ListNetworks() = %v, want %v", got, tt.want)
			}
		})
	}
}
