package datastore

import (
	"reflect"
	"testing"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/testdata"
	r "gopkg.in/rethinkdb/rethinkdb-go.v5"
)

func TestRethinkStore_FindImage(t *testing.T) {

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
			want:    &testdata.Img1,
			wantErr: false,
		},
		{
			name: "TestRethinkStore_FindImage Test 2",
			rs:   ds,
			args: args{
				id: "2",
			},
			want:    &testdata.Img2,
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
	testdata.InitMockDBData(mock)

	tests := []struct {
		name    string
		rs      *RethinkStore
		want    []metal.Image
		wantErr bool
	}{
		// Test-Data List / Test Cases:
		{
			name:    "TestRethinkStore_ListImages Test 1",
			rs:      ds,
			want:    testdata.TestImages,
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
	testdata.InitMockDBData(mock)

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
		// Test-Data List / Test Cases:
		{
			name: "TestRethinkStore_CreateImage Test 1",
			rs:   ds,
			args: args{
				i: &testdata.Img1,
			},
			want:    &testdata.Img1,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.rs.CreateImage(tt.args.i)
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.CreateImage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestRethinkStore_DeleteImage(t *testing.T) {

	// mock the DBs
	ds, mock := InitMockDB()
	testdata.InitMockDBData(mock)

	type args struct {
		img *metal.Image
	}
	tests := []struct {
		name    string
		rs      *RethinkStore
		args    args
		wantErr bool
	}{
		// Test Data Array / Test Cases:
		{
			name: "TestRethinkStore_DeleteImage Test 1",
			rs:   ds,
			args: args{
				img: &testdata.Img1,
			},
			wantErr: false,
		},
		{
			name: "TestRethinkStore_DeleteImage Test 2",
			rs:   ds,
			args: args{
				img: &testdata.Img2,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.rs.DeleteImage(tt.args.img)
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.DeleteImage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestRethinkStore_UpdateImage(t *testing.T) {

	// mock the DBs
	ds, mock := InitMockDB()
	testdata.InitMockDBData(mock)

	mock.On(r.DB("mockdb").Table("image").Get("1").Replace(func(row r.Term) r.Term {
		return r.Branch(row.Field("changed").Eq(r.Expr(testdata.Img1.Changed)), testdata.Img2, r.Error("the image was changed from another, please retry"))
	})).Return(testdata.EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("image").Get("2").Replace(func(row r.Term) r.Term {
		return r.Branch(row.Field("changed").Eq(r.Expr(testdata.Img2.Changed)), testdata.Img1, r.Error("the image was changed from another, please retry"))
	})).Return(testdata.EmptyResult, nil)

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
				&testdata.Img1, &testdata.Img2,
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
