package datastore

import (
	"context"
	"log/slog"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/generic-datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/testdata"
)

func TestRethinkStore_FindPartition(t *testing.T) {
	ds, mock := InitMockDB(t)

	ps := generic.NewDatastore(slog.Default(), ds.DBName(), ds.QueryExecutor()).Partition()
	testdata.InitMockDBData(mock)

	tests := []struct {
		name    string
		ps      generic.Storage[*metal.Partition]
		id      string
		want    *metal.Partition
		wantErr bool
	}{
		{
			name:    "Test 1",
			ps:      ps,
			id:      "1",
			want:    &testdata.Partition1,
			wantErr: false,
		},
		{
			name:    "Test 2",
			ps:      ps,
			id:      "2",
			want:    &testdata.Partition2,
			wantErr: false,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.ps.Get(context.Background(), tt.id)
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
	ps := generic.NewDatastore(slog.Default(), ds.DBName(), ds.QueryExecutor()).Partition()

	tests := []struct {
		name    string
		ps      generic.Storage[*metal.Partition]
		want    metal.Partitions
		wantErr bool
	}{
		{
			name:    "Test 1",
			ps:      ps,
			want:    testdata.TestPartitions,
			wantErr: false,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.ps.List(context.Background())
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
	ps := generic.NewDatastore(slog.Default(), ds.DBName(), ds.QueryExecutor()).Partition()

	tests := []struct {
		name    string
		ps      generic.Storage[*metal.Partition]
		p       *metal.Partition
		wantErr bool
	}{
		{
			name:    "Test 1",
			ps:      ps,
			p:       &testdata.Partition1,
			wantErr: false,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.ps.Create(context.Background(), tt.p); (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.CreatePartition() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRethinkStore_DeletePartition(t *testing.T) {
	ds, mock := InitMockDB(t)
	testdata.InitMockDBData(mock)
	ps := generic.NewDatastore(slog.Default(), ds.DBName(), ds.QueryExecutor()).Partition()

	tests := []struct {
		name    string
		ps      generic.Storage[*metal.Partition]
		p       *metal.Partition
		wantErr bool
	}{
		{
			name:    "Test 1",
			ps:      ps,
			p:       &testdata.Partition1,
			wantErr: false,
		},
		{
			name:    "Test 2",
			ps:      ps,
			p:       &testdata.Partition2,
			wantErr: false,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			err := tt.ps.Delete(context.Background(), tt.p)
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
	ps := generic.NewDatastore(slog.Default(), ds.DBName(), ds.QueryExecutor()).Partition()

	tests := []struct {
		name         string
		ps           generic.Storage[*metal.Partition]
		oldPartition *metal.Partition
		newPartition *metal.Partition
		wantErr      bool
	}{
		{
			name:         "Test 1",
			ps:           ps,
			oldPartition: &testdata.Partition1,
			newPartition: &testdata.Partition2,
			wantErr:      false,
		},
		{
			name:         "Test 2",
			ps:           ps,
			oldPartition: &testdata.Partition2,
			newPartition: &testdata.Partition1,
			wantErr:      false,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.ps.Update(context.Background(), tt.oldPartition, tt.newPartition); (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.UpdatePartition() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
