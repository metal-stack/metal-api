package datastore

import (
	"reflect"
	"testing"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/testdata"
)

func TestRethinkStore_FromHardware(t *testing.T) {
	ds, mock := InitMockDB(t)
	testdata.InitMockDBData(mock)

	tests := []struct {
		name    string
		rs      *RethinkStore
		hw      metal.MachineHardware
		want    string
		wantErr bool
	}{
		{
			name:    "determine size from machine hardware",
			rs:      ds,
			hw:      testdata.MachineHardware1,
			want:    testdata.Sz1.ID,
			wantErr: false,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.rs.FromHardware(tt.hw)
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.FromHardware() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got.ID, tt.want) {
				t.Errorf("RethinkStore.FromHardware() = %v, want %v", got.ID, tt.want)
			}
		})
	}
}
