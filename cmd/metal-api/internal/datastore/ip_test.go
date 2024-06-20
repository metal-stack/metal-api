package datastore

import (
	"context"
	"log/slog"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/generic-datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/testdata"
	"github.com/metal-stack/metal-lib/pkg/pointer"
)

func TestRethinkStore_FindIPByID(t *testing.T) {
	ds, mock := InitMockDB(t)
	testdata.InitMockDBData(mock)
	is := generic.New(slog.Default(), ds.DBName(), ds.QueryExecutor()).IP()
	tests := []struct {
		name    string
		is      generic.Storage[*metal.IP]
		id      string
		want    *metal.IP
		wantErr bool
	}{
		{
			name:    "TestRethinkStore_FindIP Test 1",
			is:      is,
			id:      "1.2.3.4",
			want:    &testdata.IP1,
			wantErr: false,
		},
		{
			name:    "TestRethinkStore_FindIP Test 2",
			is:      is,
			id:      "2.3.4.5",
			want:    &testdata.IP2,
			wantErr: false,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.is.Get(context.Background(), tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.FindIP() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("RethinkStore.FindIP() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// FIXME needs proper mock to work
func TestRethinkStore_QueryIP(t *testing.T) {
	ds, mock := InitMockDB(t)
	testdata.InitMockDBData(mock)
	is := generic.New(slog.Default(), ds.DBName(), ds.QueryExecutor()).IP()
	tests := []struct {
		name    string
		is      generic.Storage[*metal.IP]
		query   generic.EntityQuery
		want    *metal.IP
		wantErr bool
	}{
		{
			name:    "TestRethinkStore_FindIP Test 1",
			is:      is,
			query:   &IPSearchQuery{IPAddress: pointer.Pointer("1.2.3.4")},
			want:    &testdata.IP1,
			wantErr: false,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.is.Find(context.Background(), tt.query)
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.FindIP() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("RethinkStore.FindIP() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRethinkStore_ListIPs(t *testing.T) {
	ds, mock := InitMockDB(t)
	testdata.InitMockDBData(mock)
	is := generic.New(slog.Default(), ds.DBName(), ds.QueryExecutor()).IP()

	tests := []struct {
		name    string
		is      generic.Storage[*metal.IP]
		want    metal.IPs
		wantErr bool
	}{
		{
			name:    "TestRethinkStore_ListIPs Test 1",
			is:      is,
			want:    testdata.TestIPs,
			wantErr: false,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.is.List(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.ListIPs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("RethinkStore.ListIPs() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
