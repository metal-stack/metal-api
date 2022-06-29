package datastore

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/testdata"
)

func TestRethinkStore_FindPartition(t *testing.T) {
	ds, mock := InitMockDB(t)
	testdata.InitMockDBData(mock)

	tests := []struct {
		name    string
		rs      *RethinkStore
		id      string
		want    *metal.Partition
		wantErr bool
	}{
		{
			name:    "Test 1",
			rs:      ds,
			id:      "1",
			want:    &testdata.Partition1,
			wantErr: false,
		},
		{
			name:    "Test 2",
			rs:      ds,
			id:      "2",
			want:    &testdata.Partition2,
			wantErr: false,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.rs.FindPartition(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.FindPartition() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("RethinkStore.FindPartition() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRethinkStore_ListPartitions(t *testing.T) {
	ds, mock := InitMockDB(t)
	testdata.InitMockDBData(mock)

	tests := []struct {
		name    string
		rs      *RethinkStore
		want    metal.Partitions
		wantErr bool
	}{
		{
			name:    "Test 1",
			rs:      ds,
			want:    testdata.TestPartitions,
			wantErr: false,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.rs.ListPartitions()
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.ListPartitions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("RethinkStore.ListPartitions() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRethinkStore_CreatePartition(t *testing.T) {
	ds, mock := InitMockDB(t)
	testdata.InitMockDBData(mock)

	tests := []struct {
		name    string
		rs      *RethinkStore
		p       *metal.Partition
		wantErr bool
	}{
		{
			name:    "Test 1",
			rs:      ds,
			p:       &testdata.Partition1,
			wantErr: false,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.rs.CreatePartition(tt.p); (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.CreatePartition() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRethinkStore_DeletePartition(t *testing.T) {
	ds, mock := InitMockDB(t)
	testdata.InitMockDBData(mock)

	tests := []struct {
		name    string
		rs      *RethinkStore
		p       *metal.Partition
		wantErr bool
	}{
		{
			name:    "Test 1",
			rs:      ds,
			p:       &testdata.Partition1,
			wantErr: false,
		},
		{
			name:    "Test 2",
			rs:      ds,
			p:       &testdata.Partition2,
			wantErr: false,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			err := tt.rs.DeletePartition(tt.p)
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.DeletePartition() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestRethinkStore_UpdatePartition(t *testing.T) {
	ds, mock := InitMockDB(t)
	testdata.InitMockDBData(mock)

	tests := []struct {
		name         string
		rs           *RethinkStore
		oldPartition *metal.Partition
		newPartition *metal.Partition
		wantErr      bool
	}{
		{
			name:         "Test 1",
			rs:           ds,
			oldPartition: &testdata.Partition1,
			newPartition: &testdata.Partition2,
			wantErr:      false,
		},
		{
			name:         "Test 2",
			rs:           ds,
			oldPartition: &testdata.Partition2,
			newPartition: &testdata.Partition1,
			wantErr:      false,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.rs.UpdatePartition(tt.oldPartition, tt.newPartition); (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.UpdatePartition() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
