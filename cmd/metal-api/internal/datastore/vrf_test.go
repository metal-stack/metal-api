package datastore

import (
	"reflect"
	"testing"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/testdata"

	r "gopkg.in/rethinkdb/rethinkdb-go.v5"
)

func TestRethinkStore_FindVrfByProject(t *testing.T) {

	// Mock the DB:
	ds, mock := InitMockDB()
	mock.On(r.DB("mockdb").Table("vrf").Filter(r.MockAnything())).Return(testdata.Vrf1, nil)
	testdata.InitMockDBData(mock)

	type args struct {
		projectID string
		tenantID  string
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
				projectID: "p",
				tenantID:  "t",
			},
			want:    &testdata.Vrf1,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.rs.FindVrfByProject(tt.args.projectID, tt.args.tenantID)
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.FindVrfByProject() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RethinkStore.FindVrfByProject() = %v, want %v", got, tt.want)
			}
		})
	}
}
