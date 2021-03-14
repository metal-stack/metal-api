package datastore

import (
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/testdata"
)

func TestRethinkStore_FindSize(t *testing.T) {
	type args struct {
		id string
	}

	// mock the DB
	ds, mock := InitMockDB()
	testdata.InitMockDBData(mock)

	tests := []struct {
		name    string
		rs      *RethinkStore
		args    args
		want    *metal.Size
		wantErr bool
	}{
		// Test Data Array / Test Cases:
		{
			name: "TestRethinkStore_FindSize Test 1",
			rs:   ds,
			args: args{
				id: "1",
			},
			want:    &testdata.Sz1,
			wantErr: false,
		},
	}
	// Execute all tests for the test data
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.rs.FindSize(tt.args.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.FindSize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("RethinkStore.FindSize() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRethinkStore_ListSizes(t *testing.T) {
	// mock the DBs
	ds, mock := InitMockDB()
	testdata.InitMockDBData(mock)

	tests := []struct {
		name    string
		rs      *RethinkStore
		want    metal.Sizes
		wantErr bool
	}{
		// Test Data Array / Test Cases:
		{
			name:    "TestRethinkStore_ListSizes Test 1",
			rs:      ds,
			want:    testdata.TestSizes,
			wantErr: false,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.rs.ListSizes()
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.ListSizes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("RethinkStore.ListSizes() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRethinkStore_CreateSize(t *testing.T) {
	// mock the DB
	ds, mock := InitMockDB()
	testdata.InitMockDBData(mock)

	type args struct {
		size *metal.Size
	}
	tests := []struct {
		name    string
		rs      *RethinkStore
		args    args
		wantErr bool
	}{
		{
			name: "TestRethinkStore_CreateSize Test 1",
			rs:   ds,
			args: args{
				size: &testdata.Sz1,
			},
			wantErr: false,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.rs.CreateSize(tt.args.size); (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.CreateSize() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRethinkStore_DeleteSize(t *testing.T) {
	// mock the DBs
	ds, mock := InitMockDB()
	testdata.InitMockDBData(mock)

	type args struct {
		size *metal.Size
	}
	tests := []struct {
		name    string
		rs      *RethinkStore
		args    args
		wantErr bool
	}{
		// Test Data Array / Test Cases:
		{
			name: "TestRethinkStore_DeleteSize Test 1",
			rs:   ds,
			args: args{
				size: &testdata.Sz1,
			},
			wantErr: false,
		},
		{
			name: "TestRethinkStore_DeleteSize Test 2",
			rs:   ds,
			args: args{
				size: &testdata.Sz2,
			},
			wantErr: false,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			err := tt.rs.DeleteSize(tt.args.size)
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.DeleteSize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestRethinkStore_UpdateSize(t *testing.T) {
	// mock the DBs
	ds, mock := InitMockDB()
	testdata.InitMockDBData(mock)

	type args struct {
		oldSize *metal.Size
		newSize *metal.Size
	}
	tests := []struct {
		name    string
		rs      *RethinkStore
		args    args
		wantErr bool
	}{
		// Test Data Array / Test Cases:
		{
			name: "TestRethinkStore_UpdateSize Test 1",
			rs:   ds,
			args: args{
				&testdata.Sz1, &testdata.Sz2,
			},
			wantErr: false,
		},
		{
			name: "TestRethinkStore_UpdateSize Test 2",
			rs:   ds,
			args: args{
				&testdata.Sz2, &testdata.Sz1,
			},
			wantErr: false,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.rs.UpdateSize(tt.args.oldSize, tt.args.newSize); (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.UpdateSize() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRethinkStore_FromHardware(t *testing.T) {
	// mock the DBs
	ds, mock := InitMockDB()
	testdata.InitMockDBData(mock)

	type args struct {
		hw metal.MachineHardware
	}
	tests := []struct {
		name    string
		rs      *RethinkStore
		args    args
		want    string
		wantErr bool
	}{
		// Test Data Array / Test Cases:
		{
			name: "TestRethinkStore_FromHardware Test 1",
			rs:   ds,
			args: args{
				hw: testdata.MachineHardware1,
			},
			want:    testdata.Sz1.ID,
			wantErr: false,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			got, _, err := tt.rs.FromHardware(tt.args.hw)
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.FromHardware() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got.ID, tt.want) {
				t.Errorf("RethinkStore.FromHardware() = %v, want %v", got.ID, tt.want)
			}
		})
	}
}
