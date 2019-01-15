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
	//ds, mock := InitMockDB()
	//testdata.InitMockDBData(mock)

	//mock.On(r.DB("mockdb").Table("switch").Get("2")).Return(testdata.Switch2, nil)

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
		// Test Data Array / Test Cases:
		/*
			{
				name: "TestRethinkStore_FindSwitch Test 1",
				rs:   ds,
				args: args{
					id: "2",
				},
				want:    &testdata.Switch2,
				wantErr: false,
			},
		*/
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

func TestRethinkStore_findSwitchByRack(t *testing.T) {

	ds, mock := InitMockDB()

	returnSwitches := []metal.Switch{
		testdata.Switch2,
	}

	mock.On(r.DB("mockdb").Table("switch").Filter(r.MockAnything(), r.FilterOpts{})).Return(returnSwitches, nil)

	type args struct {
		rackid string
	}
	tests := []struct {
		name    string
		rs      *RethinkStore
		args    args
		want    []metal.Switch
		wantErr bool
	}{
		{
			name: "TestRethinkStore_findSwitchByRack Test 1",
			rs:   ds,
			args: args{
				rackid: "2",
			},
			want:    returnSwitches,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.rs.findSwitchByRack(tt.args.rackid)
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.findSwitchByRack() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// Because deepequal of two same objects here returns false, here are some attribute validations:
			require.Equal(t, got[0].ID, tt.want[0].ID)
			require.Equal(t, got[0].SiteID, tt.want[0].SiteID)
			require.Equal(t, got[0].RackID, tt.want[0].RackID)
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
			require.Equal(t, got[0].SiteID, tt.want[0].SiteID)
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
			got, err := tt.rs.CreateSwitch(tt.args.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.CreateSwitch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RethinkStore.CreateSwitch() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRethinkStore_DeleteSwitch(t *testing.T) {

	// mock the DBs
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
		// Test Data Array / Test Cases:

		{
			name: "TestRethinkStore_DeleteSwitch Test 1",
			rs:   ds,
			args: args{
				id: "switch1",
			},
			want:    &testdata.Switch1,
			wantErr: false,
		},
		{
			name: "TestRethinkStore_DeleteSwitch Test 2",
			rs:   ds,
			args: args{
				id: "switch2",
			},
			want:    &testdata.Switch2,
			wantErr: false,
		},

		{
			name: "TestRethinkStore_DeleteSwitch Test 3",
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
			got, err := tt.rs.DeleteSwitch(tt.args.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.DeleteSwitch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// Because deepequal of two same objects here returns false, here are some attribute validations:
			if tt.want != nil {
				require.Equal(t, got.ID, tt.want.ID)
				require.Equal(t, got.SiteID, tt.want.SiteID)
				require.Equal(t, got.RackID, tt.want.RackID)
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

func TestRethinkStore_UpdateSwitchConnections(t *testing.T) {

	// mock the DB
	ds, mock := InitMockDB()
	testdata.InitMockDBData(mock)

	type args struct {
		dev *metal.Device
	}
	tests := []struct {
		name    string
		rs      *RethinkStore
		args    args
		wantErr bool
	}{
		// Test Data Array / Test Cases:
		{
			name: "TestRethinkStore_UpdateSwitchConnections Test 1",
			rs:   ds,
			args: args{
				&testdata.D1,
			},
			wantErr: false,
		},
		{
			name: "TestRethinkStore_UpdateSwitchConnections Test 2",
			rs:   ds,
			args: args{
				&testdata.D2,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.rs.UpdateSwitchConnections(tt.args.dev); (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.UpdateSwitchConnections() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRethinkStore_findSwithcByMac(t *testing.T) {

	// mock the DB
	ds, mock := InitMockDB()

	mock.On(r.DB("mockdb").Table("switch").Filter(r.MockAnything())).Return([]metal.Switch{
		testdata.Switch1,
	}, nil)

	testdata.Switch1.FillSwitchConnections()

	type args struct {
		macs []metal.Nic
	}
	tests := []struct {
		name    string
		rs      *RethinkStore
		args    args
		want    []metal.Switch
		wantErr bool
	}{
		// Test Data Array / Test Cases:

		{
			name: "TestRethinkStore_findSwithcByMac Test 1",
			rs:   ds,
			args: args{
				macs: testdata.TestNics,
			},
			want: []metal.Switch{
				testdata.Switch1,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.rs.findSwithcByMac(tt.args.macs)
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.findSwithcByMac() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RethinkStore.findSwithcByMac() = %v, want %v", got, tt.want)
			}
		})
	}
}
