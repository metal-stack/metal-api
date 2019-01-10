package datastore

import (
	"reflect"
	"testing"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
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
			want: &RethinkStore1,
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
	metal.InitMockDBData(mock)

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
	metal.InitMockDBData(mock)

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
	metal.InitMockDBData(mock)

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
	// ds, mock := InitMockDB()
	// metal.InitMockDBData(mock)

	tests := []struct {
		name string
		rs   *RethinkStore
	}{
		/*
			{
				name: "TestRethinkStore_Connect Test 1",
				rs:   ds,
			},
		*/
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.rs.Connect()
		})
	}
}

func Test_connect(t *testing.T) {

	/*
		// mock the DB
		_, mock := InitMockDB()
		metal.InitMockDBData(mock)

		db := r.DB("mockdb")

		session, _ := r.Connect(r.ConnectOpts{
			Addresses: []string{
				"127.0.0.1",
			},
			Database: "mockdb",
			Username: "db-user",
			Password: "db-password",
			MaxIdle:  10,
			MaxOpen:  20,
		})
	*/
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
		/*
			{
				name: "Test_connect Test 1",
				args: args{
					hosts: []string{
						"db-addr",
					},
					dbname: "mockdb",
					user:   "db-user",
					pwd:    "db-password",
				},
				want:    &db,
				want1:   session,
				wantErr: true,
			},
		*/
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
	metal.InitMockDBData(mock)

	// The Connection has to be mocked here!!!
	/*
		db := r.DB("mockdb")

		session, _ := r.Connect(r.ConnectOpts{
			Addresses: []string{
				"localhost",
			},
			Database: "mockdb",
			Username: "user",
			Password: "pwd",
			MaxIdle:  10,
			MaxOpen:  20,
		})
	*/
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
		/*
			{
				name: "Test 1",
				args: args{
					hosts: []string{
						"",
					},
					dbname:   "dbname",
					username: "user",
					pwd:      "password",
				},
				want:  &db,
				want1: session,
			},
		*/
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
		/*
			{
				name: "Test 1",
				args: args{
					log: *zap.SugaredLogger,
					hosts: []string{
						"localhost",
					},
					dbname: "dbname",
					user:   "user",
					pwd:    "pwd",
				},
				want:  *r.Term,
				want1: *r.Session,
			},
		*/
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
	metal.InitMockDBData(mock)

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
	metal.InitMockDBData(mock)

	type args struct {
		opts r.TableCreateOpts
	}
	tests := []struct {
		name    string
		rs      *RethinkStore
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
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

func TestRethinkStore_table(t *testing.T) {

	// mock the DB
	ds, mock := InitMockDB()
	metal.InitMockDBData(mock)

	termimage := r.DB("mockdb").Table("image")
	termsize := r.DB("mockdb").Table("size")
	termsite := r.DB("mockdb").Table("site")
	termswitch := r.DB("mockdb").Table("switch")
	termwait := r.DB("mockdb").Table("wait")
	termipmi := r.DB("mockdb").Table("ipmi")

	type args struct {
		tablename string
	}
	tests := []struct {
		name string
		rs   *RethinkStore
		args args
		want *r.Term
	}{
		// TODO: Add test cases.
		{
			name: "TestRethinkStore_table Test 1",
			rs:   ds,
			args: args{
				tablename: "image",
			},
			want: &termimage,
		},
		{
			name: "TestRethinkStore_table Test 1",
			rs:   ds,
			args: args{
				tablename: "size",
			},
			want: &termsize,
		},
		{
			name: "TestRethinkStore_table Test 1",
			rs:   ds,
			args: args{
				tablename: "site",
			},
			want: &termsite,
		},
		{
			name: "TestRethinkStore_table Test 1",
			rs:   ds,
			args: args{
				tablename: "switch",
			},
			want: &termswitch,
		},
		{
			name: "TestRethinkStore_table Test 1",
			rs:   ds,
			args: args{
				tablename: "wait",
			},
			want: &termwait,
		},
		{
			name: "TestRethinkStore_table Test 1",
			rs:   ds,
			args: args{
				tablename: "ipmi",
			},
			want: &termipmi,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.rs.table(tt.args.tablename); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RethinkStore.table() = %v, want %v", got, tt.want)
			}
		})
	}
}
