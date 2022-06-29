package datastore

import (
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/testdata"
)

func TestRethinkStore_FindSize(t *testing.T) {
	ds, mock := InitMockDB(t)
	testdata.InitMockDBData(mock)

	tests := []struct {
		name    string
		rs      *RethinkStore
		id      string
		want    *metal.Size
		wantErr bool
	}{
		{
			name:    "TestRethinkStore_FindSize Test 1",
			rs:      ds,
			id:      "1",
			want:    &testdata.Sz1,
			wantErr: false,
		},
	}

	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.rs.FindSize(tt.id)
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
	ds, mock := InitMockDB(t)
	testdata.InitMockDBData(mock)

	tests := []struct {
		name    string
		rs      *RethinkStore
		want    metal.Sizes
		wantErr bool
	}{
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
	ds, mock := InitMockDB(t)
	testdata.InitMockDBData(mock)

	tests := []struct {
		name    string
		rs      *RethinkStore
		size    *metal.Size
		wantErr bool
	}{
		{
			name:    "TestRethinkStore_CreateSize Test 1",
			rs:      ds,
			size:    &testdata.Sz1,
			wantErr: false,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.rs.CreateSize(tt.size); (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.CreateSize() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRethinkStore_DeleteSize(t *testing.T) {
	ds, mock := InitMockDB(t)
	testdata.InitMockDBData(mock)

	tests := []struct {
		name    string
		rs      *RethinkStore
		size    *metal.Size
		wantErr bool
	}{
		{
			name:    "TestRethinkStore_DeleteSize Test 1",
			rs:      ds,
			size:    &testdata.Sz1,
			wantErr: false,
		},
		{
			name:    "TestRethinkStore_DeleteSize Test 2",
			rs:      ds,
			size:    &testdata.Sz2,
			wantErr: false,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			err := tt.rs.DeleteSize(tt.size)
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.DeleteSize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestRethinkStore_UpdateSize(t *testing.T) {
	ds, mock := InitMockDB(t)
	testdata.InitMockDBData(mock)

	tests := []struct {
		name    string
		rs      *RethinkStore
		oldSize *metal.Size
		newSize *metal.Size
		wantErr bool
	}{
		{
			name:    "TestRethinkStore_UpdateSize Test 1",
			rs:      ds,
			oldSize: &testdata.Sz1,
			newSize: &testdata.Sz2,
			wantErr: false,
		},
		{
			name:    "TestRethinkStore_UpdateSize Test 2",
			rs:      ds,
			oldSize: &testdata.Sz2,
			newSize: &testdata.Sz1,
			wantErr: false,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.rs.UpdateSize(tt.oldSize, tt.newSize); (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.UpdateSize() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRethinkStore_FromHardware(t *testing.T) {
	ds, mock := InitMockDB(t)
	testdata.InitMockDBData(mock)

	tests := []struct {
		name    string
		rs      *RethinkStore
		hw      metal.MachineHardware
		want    string
		wantErr bool
	}{
		{
			name:    "determine size from machine hardware",
			rs:      ds,
			hw:      testdata.MachineHardware1,
			want:    testdata.Sz1.ID,
			wantErr: false,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			got, _, err := tt.rs.FromHardware(tt.hw)
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
