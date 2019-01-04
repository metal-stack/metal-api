package datastore

import (
	"reflect"
	"testing"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
)

func TestRethinkStore_FindSize(t *testing.T) {

	type args struct {
		id string
	}

	// mock the DB
	ds, mock := InitMockDB()
	metal.InitMockDBData(mock)

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
			want:    &metal.Sz1,
			wantErr: false,
		},
	}
	// Execute all tests for the test data
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.rs.FindSize(tt.args.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.FindSize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RethinkStore.FindSize() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRethinkStore_ListSizes(t *testing.T) {

	// mock the DBs
	ds, mock := InitMockDB()
	metal.InitMockDBData(mock)

	tests := []struct {
		name    string
		rs      *RethinkStore
		want    []metal.Size
		wantErr bool
	}{
		// Test Data Array / Test Cases:
		{
			name:    "TestRethinkStore_ListSizes Test 1",
			rs:      ds,
			want:    metal.TestSizeArray,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.rs.ListSizes()
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.ListSizes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RethinkStore.ListSizes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRethinkStore_CreateSize(t *testing.T) {

	// mock the DB
	ds, mock := InitMockDB()
	metal.InitMockDBData(mock)

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
			name: "TestRethinkStore_CreateSize Test 1",
			rs:   ds,
			args: args{
				size: &metal.Sz1,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
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
	metal.InitMockDBData(mock)

	type args struct {
		id string
	}
	tests := []struct {
		name    string
		rs      *RethinkStore
		args    args
		want    *metal.Size
		wantErr bool
	}{
		// Test Data Array / Test Cases:
		{
			name: "TestRethinkStore_DeleteSize Test 1",
			rs:   ds,
			args: args{
				id: "1",
			},
			want:    &metal.Sz1,
			wantErr: false,
		},
		{
			name: "TestRethinkStore_DeleteSize Test 2",
			rs:   ds,
			args: args{
				id: "2",
			},
			want:    &metal.Sz2,
			wantErr: false,
		},
		{
			name: "TestRethinkStore_DeleteSize Test 3",
			rs:   ds,
			args: args{
				id: "404",
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.rs.DeleteSize(tt.args.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.DeleteSize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RethinkStore.DeleteSize() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRethinkStore_UpdateSize(t *testing.T) {

	// mock the DBs
	ds, mock := InitMockDB()
	metal.InitMockDBData(mock)

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
				&metal.Sz1, &metal.Sz2,
			},
			wantErr: false,
		},
		{
			name: "TestRethinkStore_UpdateSize Test 2",
			rs:   ds,
			args: args{
				&metal.Sz2, &metal.Sz1,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
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
	metal.InitMockDBData(mock)

	type args struct {
		hw metal.DeviceHardware
	}
	tests := []struct {
		name    string
		rs      *RethinkStore
		args    args
		want    *metal.Size
		wantErr bool
	}{
		// Test Data Array / Test Cases:
		{
			name: "TestRethinkStore_FromHardware Test 1",
			rs:   ds,
			args: args{
				hw: metal.DeviceHardware1,
			},
			want:    &metal.Sz1,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.rs.FromHardware(tt.args.hw)
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.FromHardware() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RethinkStore.FromHardware() = %v, want %v", got, tt.want)
			}
		})
	}
}
