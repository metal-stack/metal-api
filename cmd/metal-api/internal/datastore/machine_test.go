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

// Test that generates many input data
// Reference: https://golang.org/pkg/testing/quick/
func TestRethinkStore_FindMachine2(t *testing.T) {

	// Mock the DB:
	ds, mock := InitMockDB()
	testdata.InitMockDBData(mock)

	f := func(x string) bool {
		_, err := ds.FindMachine(x)
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
func TestRethinkStore_FindMachine(t *testing.T) {

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
		want    *metal.Machine
		wantErr bool
	}{
		// Test Data Array:
		{
			name: "Test 1",
			rs:   ds,
			args: args{
				id: "1",
			},
			want:    &testdata.M1,
			wantErr: false,
		},
		{
			name: "Test 2",
			rs:   ds,
			args: args{
				id: "2",
			},
			want:    &testdata.M2,
			wantErr: false,
		},
		{
			name: "Test 3",
			rs:   ds,
			args: args{
				id: "404",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Test 4",
			rs:   ds,
			args: args{
				id: "999",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Test 5",
			rs:   ds,
			args: args{
				id: "6",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Test 6",
			rs:   ds,
			args: args{
				id: "7",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Test 7",
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
			got, err := tt.rs.FindMachine(tt.args.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.FindMachine() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RethinkStore.FindMachine() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRethinkStore_SearchMachine(t *testing.T) {

	// Mock the DB:
	ds, mock := InitMockDB()
	testdata.InitMockDBData(mock)

	mock.On(r.DB("mockdb").Table("machine").Filter(func(var_1 r.Term) r.Term { return var_1.Field("macAddresses").Contains("11:11:11") })).Return([]metal.Machine{
		testdata.M1,
	}, nil)

	type args struct {
		mac string
	}
	tests := []struct {
		name    string
		rs      *RethinkStore
		args    args
		want    []metal.Machine
		wantErr bool
	}{
		// Test Data Array:
		{
			name: "Test 1",
			rs:   ds,
			args: args{
				mac: "11:11:11",
			},
			want: []metal.Machine{
				testdata.M1,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.rs.SearchMachine(tt.args.mac)
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.SearchMachine() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RethinkStore.SearchMachine() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRethinkStore_ListMachines(t *testing.T) {

	// Mock the DB:
	ds, mock := InitMockDB()
	testdata.InitMockDBData(mock)

	tests := []struct {
		name    string
		rs      *RethinkStore
		want    []metal.Machine
		wantErr bool
	}{
		// Test Data Array
		{
			name:    "Test 1",
			rs:      ds,
			want:    testdata.TestMachines,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.rs.ListMachines()
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.ListMachines() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RethinkStore.ListMachines() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRethinkStore_CreateMachine(t *testing.T) {

	// mock the DBs
	ds, mock := InitMockDB()
	testdata.InitMockDBData(mock)

	type args struct {
		d *metal.Machine
	}
	tests := []struct {
		name    string
		rs      *RethinkStore
		args    args
		wantErr bool
	}{
		// Test Data Array:
		{
			name: "Test 1",
			rs:   ds,
			args: args{
				&testdata.M4,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := tt.rs.CreateMachine(tt.args.d); (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.CreateMachine() error = %v, wantErr %v", err, tt.wantErr)
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
			name:    "Test 1",
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
			name: "Test 1",
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

func TestRethinkStore_DeleteMachine(t *testing.T) {

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
		want    *metal.Machine
		wantErr bool
	}{
		// Test Data Array:
		{
			name: "Test 1",
			rs:   ds,
			args: args{
				id: "1",
			},
			want:    &testdata.M1,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.rs.DeleteMachine(tt.args.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.DeleteMachine() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RethinkStore.DeleteMachine() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRethinkStore_UpdateMachine(t *testing.T) {

	// Mock the DB:
	ds, mock := InitMockDB()
	testdata.InitMockDBData(mock)

	type args struct {
		oldD *metal.Machine
		newD *metal.Machine
	}
	tests := []struct {
		name    string
		rs      *RethinkStore
		args    args
		wantErr bool
	}{
		// Test Data Array:
		{
			name: "Test 1",
			rs:   ds,
			args: args{
				oldD: &testdata.M1,
				newD: &testdata.M2,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.rs.UpdateMachine(tt.args.oldD, tt.args.newD); (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.UpdateMachine() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRethinkStore_AllocateMachine(t *testing.T) {

	type args struct {
		name          string
		description   string
		hostname      string
		projectid     string
		partition     *metal.Partition
		size          *metal.Size
		img           *metal.Image
		sshPubKeys    []string
		userData      string
		tenant        string
		cidrAllocator CidrAllocator
	}
	tests := []struct {
		name    string
		rs      *RethinkStore
		args    args
		want    *metal.Machine
		wantErr bool
	}{
		// Tests
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.rs.AllocateMachine(tt.args.name, tt.args.description, tt.args.hostname, tt.args.projectid, tt.args.partition, tt.args.size, tt.args.img, tt.args.sshPubKeys, tt.args.userData, tt.args.tenant, tt.args.cidrAllocator)
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.AllocateMachine() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RethinkStore.AllocateMachine() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRethinkStore_FreeMachine(t *testing.T) {

	// Mock the DB
	ds, mock := InitMockDB()
	testdata.InitMockDBData(mock)

	testdata.M2.Allocation = nil

	type args struct {
		id string
	}
	tests := []struct {
		name    string
		rs      *RethinkStore
		args    args
		want    *metal.Machine
		wantErr bool
	}{

		// Test Data Array:
		{
			name: "Test 1",
			rs:   ds,
			args: args{
				id: "2",
			},
			want:    &testdata.M2,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		fmt.Print(testdata.M4)
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.rs.FreeMachine(tt.args.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.FreeMachine() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RethinkStore.FreeMachine() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRethinkStore_RegisterMachine(t *testing.T) {

	// mock the DBs
	ds, mock := InitMockDB()
	testdata.InitMockDBData(mock)

	type args struct {
		id        string
		partition metal.Partition
		rackid    string
		sz        metal.Size
		hardware  metal.MachineHardware
		ipmi      metal.IPMI
	}
	tests := []struct {
		name    string
		rs      *RethinkStore
		args    args
		want    *metal.Machine
		wantErr bool
	}{
		// Test Data Array:
		{
			name: "Test 1",
			rs:   ds,
			args: args{
				id:        "5",
				partition: testdata.Partition1,
				rackid:    "1",
				sz:        testdata.Sz1,
				hardware:  testdata.MachineHardware1,
				ipmi:      testdata.IPMI1,
			},
			want:    &testdata.M5,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.rs.RegisterMachine(tt.args.id, tt.args.partition, tt.args.rackid, tt.args.sz, tt.args.hardware, tt.args.ipmi)
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.RegisterMachine() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RethinkStore.RegisterMachine() = %v, want %v", got, tt.want)
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

func TestRethinkStore_fillMachineList(t *testing.T) {

	// Mock the DB:
	ds, mock := InitMockDB()
	testdata.InitMockDBData(mock)

	type args struct {
		data []metal.Machine
	}
	tests := []struct {
		name    string
		rs      *RethinkStore
		args    args
		want    []metal.Machine
		wantErr bool
	}{
		// Test Data Array:
		{
			name: "Test 1",
			rs:   ds,
			args: args{
				data: []metal.Machine{
					testdata.M1, testdata.M2,
				},
			},
			want: []metal.Machine{
				testdata.M1, testdata.M2,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.rs.fillMachineList(tt.args.data...)
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.fillMachineList() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RethinkStore.fillMachineList() = %v, want %v", got, tt.want)
			}
		})
	}
}
