package datastore

import (
	"reflect"
	"testing"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
	r "gopkg.in/rethinkdb/rethinkdb-go.v5"
)

func TestRethinkStore_FindImage(t *testing.T) {

	// mock the DB
	ds, mock := InitMockDB()
	metal.InitMockDBData(mock)

	type args struct {
		id string
	}
	tests := []struct {
		name    string
		rs      *RethinkStore
		args    args
		want    *metal.Image
		wantErr bool
	}{
		// Test Data Array:
		{
			name: "TestRethinkStore_FindImage Test 1",
			rs:   ds,
			args: args{
				id: "1",
			},
			want:    &metal.Img1,
			wantErr: false,
		},
		{
			name: "TestRethinkStore_FindImage Test 2",
			rs:   ds,
			args: args{
				id: "2",
			},
			want:    &metal.Img2,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.rs.FindImage(tt.args.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.FindImage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RethinkStore.FindImage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRethinkStore_ListImages(t *testing.T) {

	// mock the DBs
	ds, mock := InitMockDB()
	metal.InitMockDBData(mock)

	tests := []struct {
		name    string
		rs      *RethinkStore
		want    []metal.Image
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name:    "TestRethinkStore_ListImages Test 1",
			rs:      ds,
			want:    metal.TestImageArray,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.rs.ListImages()
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.ListImages() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RethinkStore.ListImages() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRethinkStore_CreateImage(t *testing.T) {

	// mock the DBs
	ds, mock := InitMockDB()
	metal.InitMockDBData(mock)

	type args struct {
		i *metal.Image
	}
	tests := []struct {
		name    string
		rs      *RethinkStore
		args    args
		want    *metal.Image
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "TestRethinkStore_CreateImage Test 1",
			rs:   ds,
			args: args{
				i: &metal.Img1,
			},
			want:    &metal.Img1,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.rs.CreateImage(tt.args.i)
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.CreateImage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RethinkStore.CreateImage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRethinkStore_DeleteImage(t *testing.T) {

	// mock the DBs
	ds, mock := InitMockDB()
	metal.InitMockDBData(mock)

	type args struct {
		id string
	}
	tests := []struct {
		name    string
		rs      *RethinkStore
		args    args
		want    *metal.Image
		wantErr bool
	}{
		// Test Data Array / Test Cases:
		{
			name: "TestRethinkStore_DeleteSite Test 1",
			rs:   ds,
			args: args{
				id: "1",
			},
			want:    &metal.Img1,
			wantErr: false,
		},
		{
			name: "TestRethinkStore_DeleteSite Test 2",
			rs:   ds,
			args: args{
				id: "2",
			},
			want:    &metal.Img2,
			wantErr: false,
		},
		{
			name: "TestRethinkStore_DeleteSite Test 3",
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
			got, err := tt.rs.DeleteImage(tt.args.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.DeleteImage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RethinkStore.DeleteImage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRethinkStore_UpdateImage(t *testing.T) {

	// mock the DBs
	ds, mock := InitMockDB()
	metal.InitMockDBData(mock)

	mock.On(r.DB("mockdb").Table("image").Get("1").Replace(func(row r.Term) r.Term {
		return r.Branch(row.Field("changed").Eq(r.Expr(metal.Img1.Changed)), metal.Img2, r.Error("the image was changed from another, please retry"))
	})).Return(metal.EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("image").Get("2").Replace(func(row r.Term) r.Term {
		return r.Branch(row.Field("changed").Eq(r.Expr(metal.Img2.Changed)), metal.Img1, r.Error("the image was changed from another, please retry"))
	})).Return(metal.EmptyResult, nil)

	type args struct {
		oldImage *metal.Image
		newImage *metal.Image
	}
	tests := []struct {
		name    string
		rs      *RethinkStore
		args    args
		wantErr bool
	}{
		// Test Data Array / Test Cases:

		{
			name: "TestRethinkStore_UpdateImage Test 1",
			rs:   ds,
			args: args{
				&metal.Img1, &metal.Img2,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.rs.UpdateImage(tt.args.oldImage, tt.args.newImage); (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.UpdateImage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
