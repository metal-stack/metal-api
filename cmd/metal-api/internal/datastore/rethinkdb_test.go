package datastore

import (
	"reflect"
	"testing"

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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := New(tt.args.log, tt.args.dbhost, tt.args.dbname, tt.args.dbuser, tt.args.dbpass); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("New() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRethinkStore_initializeTables(t *testing.T) {
	type args struct {
		opts r.TableCreateOpts
	}
	tests := []struct {
		name string
		rs   *RethinkStore
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.rs.initializeTables(tt.args.opts)
		})
	}
}

func TestRethinkStore_sizeTable(t *testing.T) {
	tests := []struct {
		name string
		rs   *RethinkStore
		want *r.Term
	}{
		// TODO: Add test cases.
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
	tests := []struct {
		name string
		rs   *RethinkStore
		want *r.Term
	}{
		// TODO: Add test cases.
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
	tests := []struct {
		name string
		rs   *RethinkStore
		want *r.Term
	}{
		// TODO: Add test cases.
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
	tests := []struct {
		name string
		rs   *RethinkStore
		want *r.Term
	}{
		// TODO: Add test cases.
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
	tests := []struct {
		name string
		rs   *RethinkStore
		want *r.Term
	}{
		// TODO: Add test cases.
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
	tests := []struct {
		name string
		rs   *RethinkStore
		want *r.Term
	}{
		// TODO: Add test cases.
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
	tests := []struct {
		name string
		rs   *RethinkStore
		want *r.Term
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.rs.ipmiTable(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RethinkStore.ipmiTable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRethinkStore_db(t *testing.T) {
	tests := []struct {
		name string
		rs   *RethinkStore
		want *r.Term
	}{
		// TODO: Add test cases.
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
	tests := []struct {
		name string
		rs   *RethinkStore
		want *r.Mock
	}{
		// TODO: Add test cases.
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
	tests := []struct {
		name    string
		rs      *RethinkStore
		wantErr bool
	}{
		// TODO: Add test cases.
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
	tests := []struct {
		name string
		rs   *RethinkStore
	}{
		// TODO: Add test cases.
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
