package datastore

import (
	"log/slog"
	"reflect"
	"testing"
	"time"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/testdata"
	r "gopkg.in/rethinkdb/rethinkdb-go.v6"
)

func TestNew(t *testing.T) {
	logger := slog.Default()
	type args struct {
		log    *slog.Logger
		dbhost string
		dbname string
		dbuser string
		dbpass string
	}
	tests := []struct {
		name string
		args args
		want *RethinkStore
	}{
		{
			name: "TestNew Test 1",
			args: args{
				log:    logger,
				dbhost: "dbhost",
				dbname: "dbname",
				dbuser: "dbuser",
				dbpass: "password",
			},
			want: &RethinkStore{
				log: logger,

				dbhost: "dbhost",
				dbname: "dbname",
				dbuser: "dbuser",
				dbpass: "password",

				VRFPoolRangeMin: DefaultVRFPoolRangeMin,
				VRFPoolRangeMax: DefaultVRFPoolRangeMax,
				ASNPoolRangeMin: DefaultASNPoolRangeMin,
				ASNPoolRangeMax: DefaultASNPoolRangeMax,

				sharedMutexMaxBlockTime: 10 * time.Second,
			},
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			if got := New(tt.args.log, tt.args.dbhost, tt.args.dbname, tt.args.dbuser, tt.args.dbpass); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("New() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRethinkStore_db(t *testing.T) {
	ds, mock := InitMockDB(t)
	testdata.InitMockDBData(mock)

	theDBTerm := r.DB("mockdb")

	tests := []struct {
		name string
		rs   *RethinkStore
		want *r.Term
	}{
		{
			name: "TestRethinkStore_db Test 1",
			rs:   ds,
			want: &theDBTerm,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.rs.db(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RethinkStore.db() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRethinkStore_Close(t *testing.T) {
	ds, mock := InitMockDB(t)
	testdata.InitMockDBData(mock)

	tests := []struct {
		name    string
		rs      *RethinkStore
		wantErr bool
	}{
		{
			name:    "TestRethinkStore_Close Test 1",
			rs:      ds,
			wantErr: false,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.rs.Close(); (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.Close() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRethinkStore_sizeTable(t *testing.T) {
	ds, mock := InitMockDB(t)
	testdata.InitMockDBData(mock)

	theWantedTerm := r.DB("mockdb").Table("size")

	tests := []struct {
		name string
		rs   *RethinkStore
		want *r.Term
	}{
		{
			name: "TestRethinkStore_sizeTable Test 1",
			rs:   ds,
			want: &theWantedTerm,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.rs.sizeTable(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RethinkStore.sizeTable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRethinkStore_imageTable(t *testing.T) {
	ds, mock := InitMockDB(t)
	testdata.InitMockDBData(mock)

	theWantedTerm := r.DB("mockdb").Table("image")

	tests := []struct {
		name string
		rs   *RethinkStore
		want *r.Term
	}{
		{
			name: "TestRethinkStore_imageTable Test 1",
			rs:   ds,
			want: &theWantedTerm,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.rs.imageTable(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RethinkStore.imageTable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRethinkStore_partitionTable(t *testing.T) {
	ds, mock := InitMockDB(t)
	testdata.InitMockDBData(mock)

	theWantedTerm := r.DB("mockdb").Table("partition")

	tests := []struct {
		name string
		rs   *RethinkStore
		want *r.Term
	}{
		{
			name: "TestRethinkStore_partitionTable Test 1",
			rs:   ds,
			want: &theWantedTerm,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.rs.partitionTable(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RethinkStore.partitionTable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRethinkStore_machineTable(t *testing.T) {
	ds, mock := InitMockDB(t)
	testdata.InitMockDBData(mock)

	theWantedTerm := r.DB("mockdb").Table("machine")

	tests := []struct {
		name string
		rs   *RethinkStore
		want *r.Term
	}{
		{
			name: "Test 1",
			rs:   ds,
			want: &theWantedTerm,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.rs.machineTable(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RethinkStore.machineTable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRethinkStore_switchTable(t *testing.T) {
	ds, mock := InitMockDB(t)
	testdata.InitMockDBData(mock)

	theWantedTerm := r.DB("mockdb").Table("switch")

	tests := []struct {
		name string
		rs   *RethinkStore
		want *r.Term
	}{
		{
			name: "TestRethinkStore_switchTable Test 1",
			rs:   ds,
			want: &theWantedTerm,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.rs.switchTable(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RethinkStore.switchTable() = %v, want %v", got, tt.want)
			}
		})
	}
}
