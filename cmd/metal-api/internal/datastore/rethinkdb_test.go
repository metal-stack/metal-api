package datastore

import (
	"reflect"
	"testing"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/testdata"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"go.uber.org/zap"
	r "gopkg.in/rethinkdb/rethinkdb-go.v5"
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
		// TODO: Add test cases.
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
		// TODO: Add test cases.
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
		// TODO: Add test cases.
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
		// TODO: Add test cases.
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

func TestRethinkStore_Connect(t *testing.T) {

	// mock the DB
	_, mock := InitMockDB()
	testdata.InitMockDBData(mock)

	tests := []struct {
		name string
		rs   *RethinkStore
	}{
		// Tests
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.rs.Connect()
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
		// TODO: Add test cases.
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

func Test_mustConnect(t *testing.T) {

	// mock the DB
	_, mock := InitMockDB()
	testdata.InitMockDBData(mock)

	type args struct {
		hosts    []string
		dbname   string
		username string
		pwd      string
	}
	tests := []struct {
		name  string
		args  args
		want  *r.Term
		want1 *r.Session
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := mustConnect(tt.args.hosts, tt.args.dbname, tt.args.username, tt.args.pwd)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mustConnect() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("mustConnect() got1 = %v, want %v", got1, tt.want1)
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
		// TODO: Add test cases.
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

func TestRethinkStore_initializeTable(t *testing.T) {

	// mock the DB
	ds, mock := InitMockDB()
	testdata.InitMockDBData(mock)

	mock.On(r.DB("mockdb").TableCreate(r.MockAnything()))
	mock.On(r.DB("mockdb").TableCreate(r.MockAnything(), r.TableCreateOpts{
		Shards: 1, Replicas: 1,
	}))

	type args struct {
		table string
		opts  r.TableCreateOpts
	}
	tests := []struct {
		name    string
		rs      *RethinkStore
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "TestRethinkStore_initializeTable Test 1",
			rs:   ds,
			args: args{
				table: "size",
				opts:  r.TableCreateOpts{Shards: 1, Replicas: 1},
			},
			wantErr: false,
		},
		{
			name: "TestRethinkStore_initializeTable Test 2",
			rs:   ds,
			args: args{
				table: "size",
				opts:  r.TableCreateOpts{},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.rs.initializeTable(tt.args.table, tt.args.opts); (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.initializeTable() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRethinkStore_initializeTables(t *testing.T) {

	// mock the DB
	ds, mock := InitMockDB()
	testdata.InitMockDBData(mock)

	mock.On(r.DB("mockdb").TableCreate(r.MockAnything()))
	mock.On(r.DB("mockdb").TableCreate(r.MockAnything(), r.TableCreateOpts{
		Shards: 1, Replicas: 1,
	}))

	mock.On(r.DB("mockdb").Table("device").IndexCreate("project")).Return(r.WriteResponse{}, nil)

	type args struct {
		opts r.TableCreateOpts
	}
	tests := []struct {
		name    string
		rs      *RethinkStore
		args    args
		wantErr bool
	}{
		// Test cases:
		{
			name: "TestRethinkStore_initializeTables Test 1",
			rs:   ds,
			args: args{
				opts: r.TableCreateOpts{},
			},
			wantErr: false,
		},
		{
			name: "TestRethinkStore_initializeTables Test 2",
			rs:   ds,
			args: args{
				opts: r.TableCreateOpts{Shards: 1, Replicas: 1},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.rs.initializeTables(tt.args.opts); (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.initializeTables() error = %v, wantErr %v", err, tt.wantErr)
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

func TestRethinkStore_siteTable(t *testing.T) {

	// mock the DB
	ds, mock := InitMockDB()
	testdata.InitMockDBData(mock)

	theWantedTerm := r.DB("mockdb").Table("site")

	tests := []struct {
		name string
		rs   *RethinkStore
		want *r.Term
	}{
		// test cases:
		{
			name: "TestRethinkStore_siteTable Test 1",
			rs:   ds,
			want: &theWantedTerm,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.rs.siteTable(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RethinkStore.siteTable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRethinkStore_deviceTable(t *testing.T) {

	// mock the DB
	ds, mock := InitMockDB()
	testdata.InitMockDBData(mock)

	theWantedTerm := r.DB("mockdb").Table("device")

	tests := []struct {
		name string
		rs   *RethinkStore
		want *r.Term
	}{
		// test cases:
		{
			name: "TestRethinkStore_deviceTable Test 1",
			rs:   ds,
			want: &theWantedTerm,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.rs.deviceTable(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RethinkStore.deviceTable() = %v, want %v", got, tt.want)
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

func TestRethinkStore_ipmiTable(t *testing.T) {

	// mock the DB
	ds, mock := InitMockDB()
	testdata.InitMockDBData(mock)

	theWantedTerm := r.DB("mockdb").Table("ipmi")

	tests := []struct {
		name string
		rs   *RethinkStore
		want *r.Term
	}{
		// test cases:
		{
			name: "TestRethinkStore_ipmiTable Test 1",
			rs:   ds,
			want: &theWantedTerm,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.rs.ipmiTable(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RethinkStore.ipmiTable() = %v, want %v", got, tt.want)
			}
		})
	}
}
