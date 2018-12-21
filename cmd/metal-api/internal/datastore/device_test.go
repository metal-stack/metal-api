package datastore

import (
	"reflect"
	"testing"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
)

func TestRethinkStore_FindDevice(t *testing.T) {
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
		// TODO: Add test cases.
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
		// TODO: Add test cases.
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
	tests := []struct {
		name    string
		rs      *RethinkStore
		want    []metal.Device
		wantErr bool
	}{
		// TODO: Add test cases.
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
	type args struct {
		d *metal.Device
	}
	tests := []struct {
		name    string
		rs      *RethinkStore
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
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
		// TODO: Add test cases.
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
		// TODO: Add test cases.
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
		// TODO: Add test cases.
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
		// TODO: Add test cases.
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
		// TODO: Add test cases.
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
		// TODO: Add test cases.
	}
	for _, tt := range tests {
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
		// TODO: Add test cases.
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
		// TODO: Add test cases.
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
		// TODO: Add test cases.
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
