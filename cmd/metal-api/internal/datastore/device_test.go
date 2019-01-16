package datastore

import (
	"fmt"
	"reflect"
	"testing"
	"testing/quick"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/testdata"
	r "gopkg.in/rethinkdb/rethinkdb-go.v5"
)

// Tast that generates many input data
// Reference: https://golang.org/pkg/testing/quick/
func TestRethinkStore_FindDevice2(t *testing.T) {

	// Mock the DB:
	ds, mock := InitMockDB()
	testdata.InitMockDBData(mock)

	f := func(x string) bool {
		_, err := ds.FindDevice(x)
		returnvalue := true
		if err != nil {
			returnvalue = false
		}
		return returnvalue
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}
func TestRethinkStore_FindDevice(t *testing.T) {

	// Mock the DB:
	ds, mock := InitMockDB()
	testdata.InitMockDBData(mock)

	type args struct {
		id string
	}
	tests := []struct {
		name    string
		rs      *RethinkStore
		args    args
		want    *metal.Device
		wantErr bool
	}{
		// Test Data Array:
		{
			name: "TestRethinkStore_FindDevice Test 1",
			rs:   ds,
			args: args{
				id: "1",
			},
			want:    &testdata.D1,
			wantErr: false,
		},
		{
			name: "TestRethinkStore_FindDevice Test 2",
			rs:   ds,
			args: args{
				id: "2",
			},
			want:    &testdata.D2,
			wantErr: false,
		},
		{
			name: "TestRethinkStore_FindDevice Test 3",
			rs:   ds,
			args: args{
				id: "404",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "TestRethinkStore_FindDevice Test 4",
			rs:   ds,
			args: args{
				id: "999",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "TestRethinkStore_FindDevice Test 5",
			rs:   ds,
			args: args{
				id: "6",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "TestRethinkStore_FindDevice Test 6",
			rs:   ds,
			args: args{
				id: "7",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "TestRethinkStore_FindDevice Test 7",
			rs:   ds,
			args: args{
				id: "8",
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.rs.FindDevice(tt.args.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.FindDevice() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RethinkStore.FindDevice() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRethinkStore_SearchDevice(t *testing.T) {

	// Mock the DB:
	ds, mock := InitMockDB()
	testdata.InitMockDBData(mock)

	mock.On(r.DB("mockdb").Table("device").Filter(func(var_1 r.Term) r.Term { return var_1.Field("macAddresses").Contains("11:11:11") })).Return([]metal.Device{
		testdata.D1,
	}, nil)

	type args struct {
		mac string
	}
	tests := []struct {
		name    string
		rs      *RethinkStore
		args    args
		want    []metal.Device
		wantErr bool
	}{
		// Test Data Array:
		{
			name: "TestRethinkStore_SearchDevice Test 1",
			rs:   ds,
			args: args{
				mac: "11:11:11",
			},
			want: []metal.Device{
				testdata.D1,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.rs.SearchDevice(tt.args.mac)
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.SearchDevice() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RethinkStore.SearchDevice() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRethinkStore_ListDevices(t *testing.T) {

	// Mock the DB:
	ds, mock := InitMockDB()
	testdata.InitMockDBData(mock)

	tests := []struct {
		name    string
		rs      *RethinkStore
		want    []metal.Device
		wantErr bool
	}{
		// Test Data Array
		{
			name:    "TestRethinkStore_ListDevices Test 1",
			rs:      ds,
			want:    testdata.TestDevices,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.rs.ListDevices()
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.ListDevices() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RethinkStore.ListDevices() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRethinkStore_CreateDevice(t *testing.T) {

	// mock the DBs
	ds, mock := InitMockDB()
	testdata.InitMockDBData(mock)

	type args struct {
		d *metal.Device
	}
	tests := []struct {
		name    string
		rs      *RethinkStore
		args    args
		wantErr bool
	}{
		// Test Data Array:
		{
			name: "TestRethinkStore_CreateDevice Test 1",
			rs:   ds,
			args: args{
				&testdata.D4,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := tt.rs.CreateDevice(tt.args.d); (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.CreateDevice() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRethinkStore_FindIPMI(t *testing.T) {

	// Mock the DB:
	ds, mock := InitMockDB()
	testdata.InitMockDBData(mock)

	type args struct {
		id string
	}
	tests := []struct {
		name    string
		rs      *RethinkStore
		args    args
		want    *metal.IPMI
		wantErr bool
	}{
		// Test Data Array:
		{
			name:    "TestRethinkStore_FindIPMI Test 1",
			rs:      ds,
			args:    args{"IPMI-1"},
			want:    &testdata.IPMI1,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.rs.FindIPMI(tt.args.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.FindIPMI() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RethinkStore.FindIPMI() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRethinkStore_UpsertIPMI(t *testing.T) {

	// Mock the DB:
	ds, mock := InitMockDB()
	testdata.InitMockDBData(mock)

	type args struct {
		id   string
		ipmi *metal.IPMI
	}
	tests := []struct {
		name    string
		rs      *RethinkStore
		args    args
		wantErr bool
	}{
		// Test Data Array:
		{
			name: "TestRethinkStore_UpsertIPMI Test 1",
			rs:   ds,
			args: args{
				id:   "IPMI-1",
				ipmi: &testdata.IPMI1,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.rs.UpsertIPMI(tt.args.id, tt.args.ipmi); (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.UpsertIPMI() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRethinkStore_DeleteDevice(t *testing.T) {

	// Mock the DB:
	ds, mock := InitMockDB()
	testdata.InitMockDBData(mock)

	type args struct {
		id string
	}
	tests := []struct {
		name    string
		rs      *RethinkStore
		args    args
		want    *metal.Device
		wantErr bool
	}{
		// Test Data Array:
		{
			name: "TestRethinkStore_DeleteDevice Test 1",
			rs:   ds,
			args: args{
				id: "1",
			},
			want:    &testdata.D1,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.rs.DeleteDevice(tt.args.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.DeleteDevice() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RethinkStore.DeleteDevice() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRethinkStore_UpdateDevice(t *testing.T) {

	// Mock the DB:
	ds, mock := InitMockDB()
	testdata.InitMockDBData(mock)

	type args struct {
		oldD *metal.Device
		newD *metal.Device
	}
	tests := []struct {
		name    string
		rs      *RethinkStore
		args    args
		wantErr bool
	}{
		// Test Data Array:
		{
			name: "TestRethinkStore_UpdateDevice Test 1",
			rs:   ds,
			args: args{
				oldD: &testdata.D1,
				newD: &testdata.D2,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.rs.UpdateDevice(tt.args.oldD, tt.args.newD); (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.UpdateDevice() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRethinkStore_AllocateDevice(t *testing.T) {

	type args struct {
		name          string
		description   string
		hostname      string
		projectid     string
		site          *metal.Site
		size          *metal.Size
		img           *metal.Image
		sshPubKeys    []string
		tenant        string
		cidrAllocator CidrAllocator
	}
	tests := []struct {
		name    string
		rs      *RethinkStore
		args    args
		want    *metal.Device
		wantErr bool
	}{
		// Tests
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.rs.AllocateDevice(tt.args.name, tt.args.description, tt.args.hostname, tt.args.projectid, tt.args.site, tt.args.size, tt.args.img, tt.args.sshPubKeys, tt.args.tenant, tt.args.cidrAllocator)
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.AllocateDevice() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RethinkStore.AllocateDevice() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRethinkStore_FreeDevice(t *testing.T) {

	// Mock the DB
	ds, mock := InitMockDB()
	testdata.InitMockDBData(mock)

	testdata.D2.Allocation = nil

	type args struct {
		id string
	}
	tests := []struct {
		name    string
		rs      *RethinkStore
		args    args
		want    *metal.Device
		wantErr bool
	}{

		// Test Data Array:
		{
			name: "TestRethinkStore_FreeDevice Test 1",
			rs:   ds,
			args: args{
				id: "2",
			},
			want:    &testdata.D2,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		fmt.Print(testdata.D4)
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.rs.FreeDevice(tt.args.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.FreeDevice() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RethinkStore.FreeDevice() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRethinkStore_RegisterDevice(t *testing.T) {

	// mock the DBs
	ds, mock := InitMockDB()
	testdata.InitMockDBData(mock)

	type args struct {
		id       string
		site     metal.Site
		rackid   string
		sz       metal.Size
		hardware metal.DeviceHardware
		ipmi     metal.IPMI
	}
	tests := []struct {
		name    string
		rs      *RethinkStore
		args    args
		want    *metal.Device
		wantErr bool
	}{
		// Test Data Array:
		{
			name: "TestRethinkStore_RegisterDevice Test 1",
			rs:   ds,
			args: args{
				id:       "5",
				site:     testdata.Site1,
				rackid:   "1",
				sz:       testdata.Sz1,
				hardware: testdata.DeviceHardware1,
				ipmi:     testdata.IPMI1,
			},
			want:    &testdata.D5,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.rs.RegisterDevice(tt.args.id, tt.args.site, tt.args.rackid, tt.args.sz, tt.args.hardware, tt.args.ipmi)
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.RegisterDevice() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RethinkStore.RegisterDevice() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRethinkStore_Wait(t *testing.T) {

	type args struct {
		id    string
		alloc Allocator
	}
	tests := []struct {
		name    string
		rs      *RethinkStore
		args    args
		wantErr bool
	}{
		// Tests

	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.rs.Wait(tt.args.id, tt.args.alloc); (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.Wait() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRethinkStore_fillDeviceList(t *testing.T) {

	// Mock the DB:
	ds, mock := InitMockDB()
	testdata.InitMockDBData(mock)

	type args struct {
		data []metal.Device
	}
	tests := []struct {
		name    string
		rs      *RethinkStore
		args    args
		want    []metal.Device
		wantErr bool
	}{
		// Test Data Array:
		{
			name: "TestRethinkStore_fillDeviceList Test 1",
			rs:   ds,
			args: args{
				data: []metal.Device{
					testdata.D1, testdata.D2,
				},
			},
			want: []metal.Device{
				testdata.D1, testdata.D2,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.rs.fillDeviceList(tt.args.data...)
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.fillDeviceList() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RethinkStore.fillDeviceList() = %v, want %v", got, tt.want)
			}
		})
	}
}
