package datastore

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/testdata"
	"github.com/stretchr/testify/require"
	r "gopkg.in/rethinkdb/rethinkdb-go.v6"
)

func TestRethinkStore_FindSwitch(t *testing.T) {
	ds, mock := InitMockDB(t)
	testdata.InitMockDBData(mock)

	tests := []struct {
		name    string
		rs      *RethinkStore
		id      string
		want    *metal.Switch
		wantErr bool
	}{
		{
			name:    "TestRethinkStore_FindSwitch Test 1",
			rs:      ds,
			id:      testdata.Switch1.ID,
			want:    &testdata.Switch1,
			wantErr: false,
		},
		{
			name:    "TestRethinkStore_FindSwitch Test 2",
			rs:      ds,
			id:      "switch404",
			want:    nil,
			wantErr: true,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.rs.FindSwitch(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.FindSwitch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("RethinkStore.FindSwitch() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRethinkStore_FindSwitchByRack(t *testing.T) {
	returnSwitches := []metal.Switch{
		testdata.Switch2,
	}

	tests := []struct {
		name    string
		rackid  string
		want    []metal.Switch
		wantErr bool
	}{
		{
			name:    "TestRethinkStore_SearchSwitches Test 1 by rack id",
			rackid:  "2",
			want:    returnSwitches,
			wantErr: false,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			ds, mock := InitMockDB(t)

			mock.On(r.DB("mockdb").Table("switch").Filter(r.MockAnything(), r.FilterOpts{})).Return(returnSwitches, nil)

			got, err := ds.SearchSwitches(tt.rackid, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.SearchSwitches() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// Because deepequal of two same objects here returns false, here are some attribute validations:
			require.Equal(t, got[0].ID, tt.want[0].ID)
			require.Equal(t, got[0].PartitionID, tt.want[0].PartitionID)
			require.Equal(t, got[0].RackID, tt.want[0].RackID)

			mock.AssertExpectations(t)
		})
	}
}

func TestRethinkStore_ListSwitches(t *testing.T) {
	ds, mock := InitMockDB(t)
	testdata.InitMockDBData(mock)

	mock.On(r.DB("mockdb").Table("switch")).Return([]metal.Switch{
		testdata.Switch1, testdata.Switch2,
	}, nil)
	ds2, mock2 := InitMockDB(t)
	mock2.On(r.DB("mockdb").Table("switch")).Return([]metal.Switch{
		testdata.Switch2,
	}, nil)

	tests := []struct {
		name    string
		rs      *RethinkStore
		want    []metal.Switch
		wantErr bool
	}{
		{
			name: "TestRethinkStore_ListSwitches Test 1",
			rs:   ds,
			want: []metal.Switch{
				testdata.Switch1, testdata.Switch2,
			},
			wantErr: false,
		},
		{
			name: "TestRethinkStore_ListSwitches Test 2",
			rs:   ds2,
			want: []metal.Switch{
				testdata.Switch2,
			},
			wantErr: false,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.rs.ListSwitches()
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.ListSwitches() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// Because deepequal of two same objects here returns false, here are some attribute validations:
			require.Equal(t, got[0].ID, tt.want[0].ID)
			require.Equal(t, got[0].PartitionID, tt.want[0].PartitionID)
			require.Equal(t, got[0].RackID, tt.want[0].RackID)
		})
	}
}

func TestRethinkStore_CreateSwitch(t *testing.T) {
	ds, mock := InitMockDB(t)
	testdata.InitMockDBData(mock)

	tests := []struct {
		name    string
		rs      *RethinkStore
		s       *metal.Switch
		want    *metal.Switch
		wantErr bool
	}{
		{
			name:    "TestRethinkStore_CreateSwitch Test 1",
			rs:      ds,
			s:       &testdata.Switch1,
			want:    &testdata.Switch1,
			wantErr: false,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			err := tt.rs.CreateSwitch(tt.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.CreateSwitch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestRethinkStore_DeleteSwitch(t *testing.T) {
	ds, mock := InitMockDB(t)
	testdata.InitMockDBData(mock)

	tests := []struct {
		name    string
		rs      *RethinkStore
		s       *metal.Switch
		wantErr bool
	}{
		{
			name:    "TestRethinkStore_DeleteSwitch Test 1",
			rs:      ds,
			s:       &testdata.Switch1,
			wantErr: false,
		},
		{
			name:    "TestRethinkStore_DeleteSwitch Test 2",
			rs:      ds,
			s:       &testdata.Switch2,
			wantErr: false,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			err := tt.rs.DeleteSwitch(tt.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.DeleteSwitch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestRethinkStore_UpdateSwitch(t *testing.T) {
	ds, mock := InitMockDB(t)
	testdata.InitMockDBData(mock)

	type args struct {
		oldSwitch *metal.Switch
		newSwitch *metal.Switch
	}
	tests := []struct {
		name    string
		rs      *RethinkStore
		args    args
		wantErr bool
	}{
		{
			name: "TestRethinkStore_UpdateSwitch Test 1",
			rs:   ds,
			args: args{
				&testdata.Switch2, &testdata.Switch3,
			},
			wantErr: false,
		},
		{
			name: "TestRethinkStore_UpdateSwitch Test 2",
			rs:   ds,
			args: args{
				&testdata.Switch3, &testdata.Switch2,
			},
			wantErr: false,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.rs.UpdateSwitch(tt.args.oldSwitch, tt.args.newSwitch); (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.UpdateSwitch() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRethinkStore_FindSwitchByMac(t *testing.T) {
	tests := []struct {
		name    string
		macs    []string
		want    []metal.Switch
		wantErr bool
	}{
		{
			name: "TestRethinkStore_FindSwitch Test 1",
			macs: []string{string(testdata.Switch1.Nics[0].MacAddress)},
			want: []metal.Switch{
				testdata.Switch1,
			},
			wantErr: false,
		},
	}

	// TODO: for some reason the monotonic clock reading gets lost somewhere and a special comparer is required. find out why.
	timeComparer := cmp.Comparer(func(x, y time.Time) bool {
		return x.Unix() == y.Unix()
	})

	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			ds, mock := InitMockDB(t)
			mock.On(r.DB("mockdb").Table("switch").Filter(r.MockAnything())).Return([]metal.Switch{
				testdata.Switch1,
			}, nil)

			got, err := ds.SearchSwitches("", tt.macs)
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.SearchSwitches() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(got, tt.want, timeComparer); diff != "" {
				t.Errorf("RethinkStore.SearchSwitches() mismatch (-want +got):\n%s", diff)
			}
			mock.AssertExpectations(t)
		})
	}
}
