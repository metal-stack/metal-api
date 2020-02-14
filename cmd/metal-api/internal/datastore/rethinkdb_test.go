package datastore

import (
	"reflect"
	"testing"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/testdata"
	"github.com/metal-stack/metal-lib/zapup"
	"go.uber.org/zap"
	r "gopkg.in/rethinkdb/rethinkdb-go.v5"
)

var (
	rethinkStore1 = RethinkStore{
		SugaredLogger: zapup.MustRootLogger().Sugar(),
		dbhost:        "dbhost",
		dbname:        "dbname",
		dbuser:        "dbuser",
		dbpass:        "password",
	}
)

func TestNew(t *testing.T) {
	type args struct {
		log    *zap.Logger
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
		// Test-Data List / Test Cases:
		{
			name: "TestNew Test 1",
			args: args{
				log:    zapup.MustRootLogger(),
				dbhost: "dbhost",
				dbname: "dbname",
				dbuser: "dbuser",
				dbpass: "password",
			},
			want: &rethinkStore1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := New(tt.args.log, tt.args.dbhost, tt.args.dbname, tt.args.dbuser, tt.args.dbpass); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("New() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRethinkStore_db(t *testing.T) {
	// mock the DB
	ds, mock := InitMockDB()
	testdata.InitMockDBData(mock)

	theDBTerm := r.DB("mockdb")

	tests := []struct {
		name string
		rs   *RethinkStore
		want *r.Term
	}{
		// Test-Data List / Test Cases:
		{
			name: "TestRethinkStore_db Test 1",
			rs:   ds,
			want: &theDBTerm,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.rs.db(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RethinkStore.db() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRethinkStore_Mock(t *testing.T) {

	// mock the DB
	ds, mock := InitMockDB()
	testdata.InitMockDBData(mock)

	tests := []struct {
		name string
		rs   *RethinkStore
		want *r.Mock
	}{
		// Test-Data List / Test Cases:
		{
			name: "TestRethinkStore_Mock Test 1",
			rs:   ds,
			want: r.NewMock(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.rs.Mock(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RethinkStore.Mock() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRethinkStore_Close(t *testing.T) {

	// mock the DB
	ds, mock := InitMockDB()
	testdata.InitMockDBData(mock)

	tests := []struct {
		name    string
		rs      *RethinkStore
		wantErr bool
	}{
		// Test-Data List / Test Cases:
		{
			name:    "TestRethinkStore_Close Test 1",
			rs:      ds,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.rs.Close(); (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.Close() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_connect(t *testing.T) {

	type args struct {
		hosts  []string
		dbname string
		user   string
		pwd    string
	}
	tests := []struct {
		name    string
		args    args
		want    *r.Term
		want1   *r.Session
		wantErr bool
	}{
		// Test-Data List / Test Cases:
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := connect(tt.args.hosts, tt.args.dbname, tt.args.user, tt.args.pwd)
			if (err != nil) != tt.wantErr {
				t.Errorf("connect() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("connect() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("connect() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_retryConnect(t *testing.T) {
	type args struct {
		log    *zap.SugaredLogger
		hosts  []string
		dbname string
		user   string
		pwd    string
	}
	tests := []struct {
		name  string
		args  args
		want  *r.Term
		want1 *r.Session
	}{
		// Test-Data List / Test Cases:
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := retryConnect(tt.args.log, tt.args.hosts, tt.args.dbname, tt.args.user, tt.args.pwd)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("retryConnect() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("retryConnect() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestRethinkStore_sizeTable(t *testing.T) {

	// mock the DB
	ds, mock := InitMockDB()
	testdata.InitMockDBData(mock)

	theWantedTerm := r.DB("mockdb").Table("size")

	tests := []struct {
		name string
		rs   *RethinkStore
		want *r.Term
	}{
		// test cases:
		{
			name: "TestRethinkStore_sizeTable Test 1",
			rs:   ds,
			want: &theWantedTerm,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.rs.sizeTable(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RethinkStore.sizeTable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRethinkStore_imageTable(t *testing.T) {

	// mock the DB
	ds, mock := InitMockDB()
	testdata.InitMockDBData(mock)

	theWantedTerm := r.DB("mockdb").Table("image")

	tests := []struct {
		name string
		rs   *RethinkStore
		want *r.Term
	}{
		// test cases:
		{
			name: "TestRethinkStore_imageTable Test 1",
			rs:   ds,
			want: &theWantedTerm,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.rs.imageTable(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RethinkStore.imageTable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRethinkStore_partitionTable(t *testing.T) {

	// mock the DB
	ds, mock := InitMockDB()
	testdata.InitMockDBData(mock)

	theWantedTerm := r.DB("mockdb").Table("partition")

	tests := []struct {
		name string
		rs   *RethinkStore
		want *r.Term
	}{
		// test cases:
		{
			name: "TestRethinkStore_partitionTable Test 1",
			rs:   ds,
			want: &theWantedTerm,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.rs.partitionTable(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RethinkStore.partitionTable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRethinkStore_machineTable(t *testing.T) {

	// mock the DB
	ds, mock := InitMockDB()
	testdata.InitMockDBData(mock)

	theWantedTerm := r.DB("mockdb").Table("machine")

	tests := []struct {
		name string
		rs   *RethinkStore
		want *r.Term
	}{
		// test cases:
		{
			name: "Test 1",
			rs:   ds,
			want: &theWantedTerm,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.rs.machineTable(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RethinkStore.machineTable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRethinkStore_switchTable(t *testing.T) {

	// mock the DB
	ds, mock := InitMockDB()
	testdata.InitMockDBData(mock)

	theWantedTerm := r.DB("mockdb").Table("switch")

	tests := []struct {
		name string
		rs   *RethinkStore
		want *r.Term
	}{
		// test cases:
		{
			name: "TestRethinkStore_switchTable Test 1",
			rs:   ds,
			want: &theWantedTerm,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.rs.switchTable(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RethinkStore.switchTable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRethinkStore_waitTable(t *testing.T) {

	// mock the DB
	ds, mock := InitMockDB()
	testdata.InitMockDBData(mock)

	theWantedTerm := r.DB("mockdb").Table("wait")

	tests := []struct {
		name string
		rs   *RethinkStore
		want *r.Term
	}{
		// test cases:
		{
			name: "TestRethinkStore_waitTable Test 1",
			rs:   ds,
			want: &theWantedTerm,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.rs.waitTable(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RethinkStore.waitTable() = %v, want %v", got, tt.want)
			}
		})
	}
}
