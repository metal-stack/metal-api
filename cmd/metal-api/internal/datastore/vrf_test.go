
package datastore

import (
	"reflect"
	"testing"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/testdata"
	r "gopkg.in/rethinkdb/rethinkdb-go.v5"
)
func TestRethinkStore_FindVrf(t *testing.T) {

	// Mock the DB:
	ds, mock := InitMockDB()
	testdata.InitMockDBData(mock)

	mock.On(r.DB("mockdb").Table("vrf").Filter(r.MockAnything())).Return(testdata.Vrf1, nil)

	type args struct {
		f map[string]interface{}
	}
	tests := []struct {
		name    string
		rs      *RethinkStore
		args    args
		want    *metal.Vrf
		wantErr bool
	}{
		// Test Data Array:
		{
			name: "Test find Vrf1 by tenant and projectid",
			rs:   ds,
			args: args{
				f: map[string]interface{}{"tenant": "t", "projectid": "p"},
			},
			want:    &testdata.Vrf1,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.rs.FindVrf(tt.args.f)
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.FindVrf() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RethinkStore.FindVrf() = %v, want %v", got, tt.want)
			}
		})
	}
}
