package datastore

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/testdata"
)

func TestRethinkStore_FindPartition(t *testing.T) {

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
		want    *metal.Partition
		wantErr bool
	}{
		// Test Data Array:
		{
			name: "Test 1",
			rs:   ds,
			args: args{
				id: "1",
			},
			want:    &testdata.Partition1,
			wantErr: false,
		},
		{
			name: "Test 2",
			rs:   ds,
			args: args{
				id: "2",
			},
			want:    &testdata.Partition2,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.rs.FindPartition(tt.args.id)
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

	// mock the DBs
	ds, mock := InitMockDB()
	testdata.InitMockDBData(mock)

	tests := []struct {
		name    string
		rs      *RethinkStore
		want    metal.Partitions
		wantErr bool
	}{
		// Test Data Array / Test Cases:
		{
			name:    "Test 1",
			rs:      ds,
			want:    testdata.TestPartitions,
			wantErr: false,
		},
	}
	for _, tt := range tests {
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

	// mock the DB
	ds, mock := InitMockDB()
	testdata.InitMockDBData(mock)

	type args struct {
		part *metal.Partition
	}
	tests := []struct {
		name    string
		rs      *RethinkStore
		args    args
		wantErr bool
	}{
		// Test Data Array / Test Cases:
		{
			name: "Test 1",
			rs:   ds,
			args: args{
				part: &testdata.Partition1,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.rs.CreatePartition(tt.args.part); (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.CreatePartition() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRethinkStore_DeletePartition(t *testing.T) {

	// mock the DBs
	ds, mock := InitMockDB()
	testdata.InitMockDBData(mock)

	type args struct {
		p *metal.Partition
	}
	tests := []struct {
		name    string
		rs      *RethinkStore
		args    args
		wantErr bool
	}{
		// Test Data Array / Test Cases:
		{
			name: "Test 1",
			rs:   ds,
			args: args{
				p: &testdata.Partition1,
			},
			wantErr: false,
		},
		{
			name: "Test 2",
			rs:   ds,
			args: args{
				p: &testdata.Partition2,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.rs.DeletePartition(tt.args.p)
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.DeletePartition() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestRethinkStore_UpdatePartition(t *testing.T) {

	// mock the DBs
	ds, mock := InitMockDB()
	testdata.InitMockDBData(mock)

	type args struct {
		oldF *metal.Partition
		newF *metal.Partition
	}
	tests := []struct {
		name    string
		rs      *RethinkStore
		args    args
		wantErr bool
	}{
		// Test Data Array / Test Cases:
		{
			name: "Test 1",
			rs:   ds,
			args: args{
				&testdata.Partition1, &testdata.Partition2,
			},
			wantErr: false,
		},
		{
			name: "Test 2",
			rs:   ds,
			args: args{
				&testdata.Partition2, &testdata.Partition1,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.rs.UpdatePartition(tt.args.oldF, tt.args.newF); (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.UpdatePartition() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
