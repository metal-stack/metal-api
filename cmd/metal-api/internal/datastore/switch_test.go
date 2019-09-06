package datastore

import (
	"reflect"
	"testing"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/testdata"
	"github.com/stretchr/testify/require"
	r "gopkg.in/rethinkdb/rethinkdb-go.v5"
)

func TestRethinkStore_FindSwitch(t *testing.T) {
	// mock the DB
	ds, mock := InitMockDB()
	testdata.InitMockDBData(mock)

	type args struct {
		id string
	}
	tests := []struct {
		name    string
		rs      *RethinkStore
		args    args
		want    *metal.Switch
		wantErr bool
	}{
		{
			name: "TestRethinkStore_FindSwitch Test 1",
			rs:   ds,
			args: args{
				id: testdata.Switch1.ID,
			},
			want:    &testdata.Switch1,
			wantErr: false,
		},
		{
			name: "TestRethinkStore_FindSwitch Test 2",
			rs:   ds,
			args: args{
				id: "switch404",
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.rs.FindSwitch(tt.args.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.FindSwitch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RethinkStore.FindSwitch() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRethinkStore_FindSwitchByRack(t *testing.T) {

	returnSwitches := []metal.Switch{
		testdata.Switch2,
	}

	type args struct {
		rackid string
	}
	tests := []struct {
		name    string
		args    args
		want    []metal.Switch
		wantErr bool
	}{
		{
			name: "TestRethinkStore_SearchSwitches Test 1 by rack id",
			args: args{
				rackid: "2",
			},
			want:    returnSwitches,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ds, mock := InitMockDB()

			mock.On(r.DB("mockdb").Table("switch").Filter(r.MockAnything(), r.FilterOpts{})).Return(returnSwitches, nil)

			got, err := ds.SearchSwitches(tt.args.rackid, nil)
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

	// mock the DBs
	ds, mock := InitMockDB()
	testdata.InitMockDBData(mock)

	mock.On(r.DB("mockdb").Table("switch")).Return([]metal.Switch{
		testdata.Switch1, testdata.Switch2,
	}, nil)
	ds2, mock2 := InitMockDB()
	mock2.On(r.DB("mockdb").Table("switch")).Return([]metal.Switch{
		testdata.Switch2,
	}, nil)

	tests := []struct {
		name    string
		rs      *RethinkStore
		want    []metal.Switch
		wantErr bool
	}{
		// Test Data Array / Test Cases:

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
	for _, tt := range tests {
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

	// mock the DB
	ds, mock := InitMockDB()
	testdata.InitMockDBData(mock)

	type args struct {
		s *metal.Switch
	}
	tests := []struct {
		name    string
		rs      *RethinkStore
		args    args
		want    *metal.Switch
		wantErr bool
	}{
		// Test Data Array / Test Cases:
		{
			name: "TestRethinkStore_CreateSwitch Test 1",
			rs:   ds,
			args: args{
				s: &testdata.Switch1,
			},
			want:    &testdata.Switch1,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.rs.CreateSwitch(tt.args.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.CreateSwitch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestRethinkStore_DeleteSwitch(t *testing.T) {

	// mock the DBs
	ds, mock := InitMockDB()
	testdata.InitMockDBData(mock)

	tests := []struct {
		name    string
		rs      *RethinkStore
		s       *metal.Switch
		wantErr bool
	}{
		// Test Data Array / Test Cases:

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
	for _, tt := range tests {
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

	// mock the DB
	ds, mock := InitMockDB()
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
		// Test Data Array / Test Cases:
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
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.rs.UpdateSwitch(tt.args.oldSwitch, tt.args.newSwitch); (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.UpdateSwitch() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRethinkStore_FindSwitchByMac(t *testing.T) {
	type args struct {
		macs []string
	}
	tests := []struct {
		name    string
		args    args
		want    []metal.Switch
		wantErr bool
	}{
		{
			name: "TestRethinkStore_FindSwitch Test 1",
			args: args{
				macs: []string{string(testdata.Switch1.Nics[0].MacAddress)},
			},
			want: []metal.Switch{
				testdata.Switch1,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ds, mock := InitMockDB()
			mock.On(r.DB("mockdb").Table("switch").Filter(r.MockAnything())).Return([]metal.Switch{
				testdata.Switch1,
			}, nil)

			got, err := ds.SearchSwitches("", tt.args.macs)
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.SearchSwitches() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RethinkStore.SearchSwitches() = %v, want %v", got, tt.want)
			}
			mock.AssertExpectations(t)
		})
	}
}
