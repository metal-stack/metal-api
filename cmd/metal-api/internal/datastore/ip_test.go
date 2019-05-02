package datastore

import (
	"reflect"
	"testing"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/testdata"
)

func TestRethinkStore_FindIP(t *testing.T) {

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
		want    *metal.IP
		wantErr bool
	}{
		// Test Data Array:
		{
			name: "TestRethinkStore_FindIP Test 1",
			rs:   ds,
			args: args{
				id: "1.2.3.4",
			},
			want:    &testdata.IP1,
			wantErr: false,
		},
		{
			name: "TestRethinkStore_FindIP Test 2",
			rs:   ds,
			args: args{
				id: "2.3.4.5",
			},
			want:    &testdata.IP2,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.rs.FindIP(tt.args.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.FindIP() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RethinkStore.FindIP() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRethinkStore_ListIPs(t *testing.T) {

	// mock the DBs
	ds, mock := InitMockDB()
	testdata.InitMockDBData(mock)

	tests := []struct {
		name    string
		rs      *RethinkStore
		want    []metal.IP
		wantErr bool
	}{
		// Test-Data List / Test Cases:
		{
			name:    "TestRethinkStore_ListIPs Test 1",
			rs:      ds,
			want:    testdata.TestIPs,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.rs.ListIPs()
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.ListIPs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RethinkStore.ListIPs() = %v, want %v", got, tt.want)
			}
		})
	}
}
