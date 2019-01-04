package datastore

import (
	"fmt"
	"reflect"
	"testing"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
)

func TestRethinkStore_FindDevice(t *testing.T) {

	// Mock the DB:
	ds, mock := InitMockDB()
	metal.InitMockDBData(mock)

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
			want:    &metal.D1,
			wantErr: false,
		},
		{
			name: "TestRethinkStore_FindDevice Test 2",
			rs:   ds,
			args: args{
				id: "2",
			},
			want:    &metal.D2,
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
	metal.InitMockDBData(mock)

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
				metal.D1,
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
	metal.InitMockDBData(mock)

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
			want:    metal.TestDeviceArray,
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
	metal.InitMockDBData(mock)

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
				&metal.D4,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.rs.CreateDevice(tt.args.d); (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.CreateDevice() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRethinkStore_FindIPMI(t *testing.T) {

	// Mock the DB:
	ds, mock := InitMockDB()
	metal.InitMockDBData(mock)

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
			want:    &metal.IPMI1,
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
	metal.InitMockDBData(mock)

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
				ipmi: &metal.IPMI1,
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
	metal.InitMockDBData(mock)

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
			want:    &metal.D1,
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
	metal.InitMockDBData(mock)

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
				oldD: &metal.D1,
				newD: &metal.D2,
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

	/*
		// Mock the DB
		ds, mock := InitMockDB()
		metal.InitMockDBData(mock)

		mock.On(r.DB("mockdb").Table("device").Get("1").Replace(func(row r.Term) r.Term {
			return r.Branch(row.Field("changed").Eq(r.Expr(metal.D1.Changed)), metal.D2, r.Error("the device was changed from another, please retry"))
		})).Return(metal.EmptyResult, nil)

		mock.On(r.DB("mockdb").Table("size").Get("1")).Return(metal.Sz1, nil)
		mock.On(r.DB("mockdb").Table("site").Get("1")).Return(metal.Site1, nil)
		mock.On(r.DB("mockdb").Table("image").Get("1")).Return(metal.Img1, nil)
		mock.On(r.DB("mockdb").Table("device").Get("1").Delete()).Return(metal.EmptyResult, nil)
	*/
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
		/*
			{
				name: "Test 1",
				rs:   ds,
				args: args{
					name:        "name",
					description: "description",
					hostname:    "hostname",
					projectid:   "projectid",
					site:        &metal.Site1,
					size:        &metal.Sz1,
					img:         &metal.Img1,
					sshPubKeys: []string{
						"ssh:123", "ssh:321",
					},
					tenant:        "tenant",
					cidrAllocator: CidrAllocator{
						Allocate(uuid, tenant, project, name, description, os string) (string, error),
						Release(uuid string) error,
					},
				},
				want:    &metal.D1,
				wantErr: false,
			},
		*/
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

	//&{{2   0001-01-01 00:00:00 +0000 UTC 0001-01-01 00:00:00 +0000 UTC} {{1 site1 description 1 0001-01-01 00:00:00 +0000 UTC 0001-01-01 00:00:00 +0000 UTC}} 1  0xc00011a280 1 {0 0 [] []} <nil>}
	//&{{2   0001-01-01 00:00:00 +0000 UTC 0001-01-01 00:00:00 +0000 UTC} {{1 site1 description 1 0001-01-01 00:00:00 +0000 UTC 0001-01-01 00:00:00 +0000 UTC}} 1  0xc62240 1 {0 0 [] []} 0xc62d20}

	// Mock the DB
	//ds, mock := InitMockDB()
	//metal.InitMockDBData(mock)

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
		/*
			// Test Data Array:
				{
					name: "TestRethinkStore_FreeDevice Test 1",
					rs:   ds,
					args: args{
						id: "2",
					},
					want:    &metal.D2,
					wantErr: false,
				},
		*/
	}
	for _, tt := range tests {
		fmt.Print(metal.D4)
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
	metal.InitMockDBData(mock)

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
				site:     metal.Site1,
				rackid:   "1",
				sz:       metal.Sz1,
				hardware: metal.DeviceHardware1,
				ipmi:     metal.IPMI1,
			},
			want:    &metal.D5,
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

	/*
			// mock the DBs
			ds, mock := InitMockDB()
			metal.InitMockDBData(mock)

		mock.On(r.DB("mockdb").Table("device").Get("1").Replace(func(row r.Term) r.Term {
			return r.Branch(row.Field("changed").Eq(r.Expr(metal.D1.Changed)), metal.D2, r.Error("the device was changed from another, please retry"))
		})).Return(metal.EmptyResult, nil)
	*/

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
		/*
			// Test Data Array:
			{
				name: "TestRethinkStore_Wait Test 1",
				rs:   ds,
				args: args{
					id:    "1",
					alloc: nil,
				},
				wantErr: false,
			},
		*/
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
	metal.InitMockDBData(mock)

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
					metal.D1, metal.D2,
				},
			},
			want: []metal.Device{
				metal.D1, metal.D2,
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
