package datastore

import (
	"log/slog"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/testdata"
	"github.com/metal-stack/metal-lib/pkg/testcommon"
)

func TestNew(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
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

				sharedMutexCheckInterval: defaultSharedMutexCheckInterval,
			},
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			got := New(tt.args.log, tt.args.dbhost, tt.args.dbname, tt.args.dbuser, tt.args.dbpass)
			if diff := cmp.Diff(got, tt.want, testcommon.IgnoreUnexported(), cmpopts.IgnoreTypes(slog.Logger{})); diff != "" {
				t.Errorf("New() mismatch (-want +got):\n%s", diff)
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
