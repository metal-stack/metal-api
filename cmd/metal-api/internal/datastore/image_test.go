package datastore

import (
	"reflect"
	"testing"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/google/go-cmp/cmp"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/testdata"
	"github.com/stretchr/testify/assert"
	r "gopkg.in/rethinkdb/rethinkdb-go.v6"
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
				id: "image-1",
			},
			want:    &testdata.Img1,
			wantErr: false,
		},
		{
			name: "TestRethinkStore_FindImage Test 2",
			rs:   ds,
			args: args{
				id: "image-2",
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
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("RethinkStore.FindImage() mismatch (-want +got):\n%s", diff)
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
		want    metal.Images
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
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("RethinkStore.ListImages() mismatch (-want +got):\n%s", diff)
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
func Test_getMostRecentImageFor(t *testing.T) {
	invalid := time.Now().Add(time.Hour * -1)
	valid := time.Now().Add(time.Hour)
	ubuntu14_1 := metal.Image{Base: metal.Base{ID: "ubuntu-14.1"}, OS: "ubuntu", Version: "14.1", ExpirationDate: valid}
	ubuntu14_04 := metal.Image{Base: metal.Base{ID: "ubuntu-14.04"}, OS: "ubuntu", Version: "14.04", ExpirationDate: valid}
	ubuntu17_04 := metal.Image{Base: metal.Base{ID: "ubuntu-17.04"}, OS: "ubuntu", Version: "17.04", ExpirationDate: valid}
	ubuntu17_10 := metal.Image{Base: metal.Base{ID: "ubuntu-17.10"}, OS: "ubuntu", Version: "17.10", ExpirationDate: valid}
	ubuntu18_04 := metal.Image{Base: metal.Base{ID: "ubuntu-18.04"}, OS: "ubuntu", Version: "18.04", ExpirationDate: valid}
	ubuntu19_04 := metal.Image{Base: metal.Base{ID: "ubuntu-19.04"}, OS: "ubuntu", Version: "19.4", ExpirationDate: valid}
	ubuntu19_10 := metal.Image{Base: metal.Base{ID: "ubuntu-19.10"}, OS: "ubuntu", Version: "19.10", ExpirationDate: valid}
	ubuntu20_04_20200401 := metal.Image{Base: metal.Base{ID: "ubuntu-20.04.20200401"}, OS: "ubuntu", Version: "20.04.20200401", ExpirationDate: valid}
	ubuntu20_04_20200501 := metal.Image{Base: metal.Base{ID: "ubuntu-20.04.20200501"}, OS: "ubuntu", Version: "20.04.20200501", ExpirationDate: valid}
	ubuntu20_04_20200502 := metal.Image{Base: metal.Base{ID: "ubuntu-20.04.20200502"}, OS: "ubuntu", Version: "20.04.20200502", ExpirationDate: valid}
	ubuntu20_04_20200603 := metal.Image{Base: metal.Base{ID: "ubuntu-20.04.20200603"}, OS: "ubuntu", Version: "20.04.20200603", ExpirationDate: valid}
	ubuntu20_04_20200602 := metal.Image{Base: metal.Base{ID: "ubuntu-20.10.20200602"}, OS: "ubuntu", Version: "20.10.20200602", ExpirationDate: valid}
	ubuntu20_10_20200603 := metal.Image{Base: metal.Base{ID: "ubuntu-20.10.20200603"}, OS: "ubuntu", Version: "20.10.20200603", ExpirationDate: invalid}

	alpine3_9 := metal.Image{Base: metal.Base{ID: "alpine-3.9"}, OS: "alpine", Version: "3.9", ExpirationDate: valid}
	alpine3_9_20191012 := metal.Image{Base: metal.Base{ID: "alpine-3.9.20191012"}, OS: "alpine", Version: "3.9.20191012", ExpirationDate: valid}
	alpine3_10 := metal.Image{Base: metal.Base{ID: "alpine-3.10"}, OS: "alpine", Version: "3.10", ExpirationDate: valid}
	alpine3_10_20191012 := metal.Image{Base: metal.Base{ID: "alpine-3.10.20191012"}, OS: "alpine", Version: "3.10.20191012", ExpirationDate: valid}
	tests := []struct {
		name    string
		id      string
		images  []metal.Image
		want    *metal.Image
		wantErr bool
	}{
		{
			name:    "simple",
			id:      "ubuntu-19.04",
			images:  []metal.Image{ubuntu20_04_20200502, ubuntu19_10, ubuntu17_04, ubuntu20_04_20200401, ubuntu19_04, ubuntu14_1, ubuntu20_04_20200501, ubuntu18_04, ubuntu14_04, ubuntu17_10, ubuntu20_04_20200603},
			want:    &ubuntu19_04,
			wantErr: false,
		},
		{
			name:    "also simple",
			id:      "ubuntu-19.10",
			images:  []metal.Image{ubuntu20_04_20200502, alpine3_9, ubuntu19_10, ubuntu17_04, ubuntu20_04_20200401, ubuntu19_04, ubuntu14_1, ubuntu20_04_20200501, ubuntu18_04, ubuntu14_04, ubuntu17_10, ubuntu20_04_20200603, alpine3_9_20191012},
			want:    &ubuntu19_10,
			wantErr: false,
		},
		{
			name:    "patch given with no match",
			id:      "ubuntu-20.04.2020",
			images:  []metal.Image{ubuntu20_04_20200502, alpine3_9, ubuntu19_10, ubuntu17_04, ubuntu20_04_20200401, ubuntu19_04, ubuntu14_1, ubuntu20_04_20200501, ubuntu18_04, ubuntu14_04, ubuntu17_10, ubuntu20_04_20200603, alpine3_9_20191012},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "patch given with match",
			id:      "ubuntu-20.04.20200502",
			images:  []metal.Image{ubuntu20_04_20200502, alpine3_9, ubuntu19_10, ubuntu17_04, ubuntu20_04_20200401, ubuntu19_04, ubuntu14_1, ubuntu20_04_20200501, ubuntu18_04, ubuntu14_04, ubuntu17_10, ubuntu20_04_20200603, alpine3_9_20191012},
			want:    &ubuntu20_04_20200502,
			wantErr: false,
		},
		{
			name:    "alpine",
			id:      "alpine-3.10",
			images:  []metal.Image{ubuntu20_04_20200502, alpine3_9, ubuntu19_10, ubuntu17_04, alpine3_10_20191012, ubuntu20_04_20200401, ubuntu19_04, ubuntu14_1, ubuntu20_04_20200501, ubuntu18_04, alpine3_10, ubuntu14_04, ubuntu17_10, ubuntu20_04_20200603, alpine3_9_20191012},
			want:    &alpine3_10_20191012,
			wantErr: false,
		},
		{
			name:    "alpine II",
			id:      "alpine-3.9",
			images:  []metal.Image{ubuntu20_04_20200502, alpine3_9, ubuntu19_10, ubuntu17_04, alpine3_10_20191012, ubuntu20_04_20200401, ubuntu19_04, ubuntu14_1, ubuntu20_04_20200501, ubuntu18_04, alpine3_10, ubuntu14_04, ubuntu17_10, ubuntu20_04_20200603, alpine3_9_20191012},
			want:    &alpine3_9_20191012,
			wantErr: false,
		},
		{
			name:    "ubuntu with invalid",
			id:      "ubuntu-20.10",
			images:  []metal.Image{ubuntu20_04_20200602, ubuntu20_10_20200603},
			want:    &ubuntu20_04_20200602,
			wantErr: false,
		},
	}
	rs := &RethinkStore{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := rs.getMostRecentImageFor(tt.id, tt.images)
			if (err != nil) != tt.wantErr {
				t.Errorf("getMostRecentImageFor() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getMostRecentImageFor() %s\n", cmp.Diff(got, tt.want))
			}
		})
	}
}

func Test_sortImages(t *testing.T) {
	ubuntu14_1 := metal.Image{Base: metal.Base{ID: "ubuntu-14.1"}, OS: "ubuntu", Version: "14.1"}
	ubuntu14_04 := metal.Image{Base: metal.Base{ID: "ubuntu-14.04"}, OS: "ubuntu", Version: "14.04"}
	ubuntu17_04 := metal.Image{Base: metal.Base{ID: "ubuntu-17.04"}, OS: "ubuntu", Version: "17.04"}
	ubuntu17_10 := metal.Image{Base: metal.Base{ID: "ubuntu-17.10"}, OS: "ubuntu", Version: "17.10"}
	ubuntu18_04 := metal.Image{Base: metal.Base{ID: "ubuntu-18.04"}, OS: "ubuntu", Version: "18.04"}
	ubuntu19_04 := metal.Image{Base: metal.Base{ID: "ubuntu-19.04"}, OS: "ubuntu", Version: "19.04"}
	ubuntu19_10 := metal.Image{Base: metal.Base{ID: "ubuntu-19.10"}, OS: "ubuntu", Version: "19.10"}
	ubuntu20_04_20200401 := metal.Image{Base: metal.Base{ID: "ubuntu-20.04.20200401"}, OS: "ubuntu", Version: "20.04.20200401"}
	ubuntu20_04_20200501 := metal.Image{Base: metal.Base{ID: "ubuntu-20.04.20200501"}, OS: "ubuntu", Version: "20.04.20200501"}
	ubuntu20_04_20200502 := metal.Image{Base: metal.Base{ID: "ubuntu-20.04.20200502"}, OS: "ubuntu", Version: "20.04.20200502"}
	ubuntu20_04_20200603 := metal.Image{Base: metal.Base{ID: "ubuntu-20.04.20200603"}, OS: "ubuntu", Version: "20.04.20200603"}

	alpine3_9 := metal.Image{Base: metal.Base{ID: "alpine-3.9"}, OS: "alpine", Version: "3.9"}
	alpine3_10 := metal.Image{Base: metal.Base{ID: "alpine-3.10"}, OS: "alpine", Version: "3.10"}

	debian17_04 := metal.Image{Base: metal.Base{ID: "debian-17.04"}, OS: "debian", Version: "17.04"}
	debian17_10 := metal.Image{Base: metal.Base{ID: "debian-17.10"}, OS: "debian", Version: "17.10"}

	tests := []struct {
		name   string
		images []metal.Image
		want   []metal.Image
	}{
		{
			name:   "ubuntu versions",
			images: []metal.Image{ubuntu20_04_20200502, ubuntu19_10, ubuntu17_04, ubuntu20_04_20200401, ubuntu19_04, ubuntu14_1, ubuntu20_04_20200501, ubuntu18_04, ubuntu14_04, ubuntu17_10, ubuntu20_04_20200603},
			want:   []metal.Image{ubuntu20_04_20200603, ubuntu20_04_20200502, ubuntu20_04_20200501, ubuntu20_04_20200401, ubuntu19_10, ubuntu19_04, ubuntu18_04, ubuntu17_10, ubuntu17_04, ubuntu14_04, ubuntu14_1},
		},
		{
			name:   "ubuntu and alpine versions",
			images: []metal.Image{ubuntu20_04_20200502, alpine3_9, ubuntu19_10, ubuntu17_04, ubuntu20_04_20200401, ubuntu19_04, ubuntu14_1, ubuntu20_04_20200501, ubuntu18_04, ubuntu14_04, ubuntu17_10, ubuntu20_04_20200603, alpine3_10},
			want:   []metal.Image{alpine3_10, alpine3_9, ubuntu20_04_20200603, ubuntu20_04_20200502, ubuntu20_04_20200501, ubuntu20_04_20200401, ubuntu19_10, ubuntu19_04, ubuntu18_04, ubuntu17_10, ubuntu17_04, ubuntu14_04, ubuntu14_1},
		},
		{
			name:   "ubuntu and artificial debian",
			images: []metal.Image{ubuntu20_04_20200502, ubuntu19_10, ubuntu17_04, ubuntu20_04_20200401, ubuntu19_04, ubuntu14_1, ubuntu20_04_20200501, debian17_10, ubuntu18_04, ubuntu14_04, ubuntu17_10, ubuntu20_04_20200603, debian17_04},
			want:   []metal.Image{debian17_10, debian17_04, ubuntu20_04_20200603, ubuntu20_04_20200502, ubuntu20_04_20200501, ubuntu20_04_20200401, ubuntu19_10, ubuntu19_04, ubuntu18_04, ubuntu17_10, ubuntu17_04, ubuntu14_04, ubuntu14_1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sortImages(tt.images); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("sortImages() \n%s", cmp.Diff(got, tt.want))
			}
		})
	}
}

func TestSemver(t *testing.T) {
	c, err := semver.NewConstraint("~1")
	assert.NoError(t, err)
	assert.NotNil(t, c)
	v, err := semver.NewVersion("1.99.99")
	assert.NoError(t, err)
	assert.NotNil(t, v)
	satisfies := c.Check(v)
	assert.True(t, satisfies)
	v, err = semver.StrictNewVersion("19.01")
	assert.Error(t, err)
	assert.Nil(t, v)
}
func TestRethinkStore_DeleteOrphanImages(t *testing.T) {
	ds, mock := InitMockDB()
	testdata.InitMockDBData(mock)

	invalid := time.Now().Add(time.Hour * -1)
	valid := time.Now().Add(time.Hour)
	ubuntu14_1 := metal.Image{Base: metal.Base{ID: "ubuntu-14.1"}, OS: "ubuntu", Version: "14.1", ExpirationDate: valid}
	ubuntu14_04 := metal.Image{Base: metal.Base{ID: "ubuntu-14.04"}, OS: "ubuntu", Version: "14.04", ExpirationDate: valid}
	ubuntu17_04 := metal.Image{Base: metal.Base{ID: "ubuntu-17.04"}, OS: "ubuntu", Version: "17.04", ExpirationDate: valid}
	ubuntu17_10 := metal.Image{Base: metal.Base{ID: "ubuntu-17.10"}, OS: "ubuntu", Version: "17.10", ExpirationDate: valid}
	ubuntu18_04 := metal.Image{Base: metal.Base{ID: "ubuntu-18.04"}, OS: "ubuntu", Version: "18.04", ExpirationDate: valid}
	ubuntu19_04 := metal.Image{Base: metal.Base{ID: "ubuntu-19.04"}, OS: "ubuntu", Version: "19.04", ExpirationDate: invalid} // not allocated
	ubuntu19_10 := metal.Image{Base: metal.Base{ID: "ubuntu-19.10"}, OS: "ubuntu", Version: "19.10", ExpirationDate: invalid} // allocated
	alpine3_9 := metal.Image{Base: metal.Base{ID: "alpine-3.9"}, OS: "alpine", Version: "3.9", ExpirationDate: invalid}       // not allocated
	alpine3_10 := metal.Image{Base: metal.Base{ID: "alpine-3.10"}, OS: "alpine", Version: "3.10", ExpirationDate: invalid}    // not allocated but kept because last from that os
	tests := []struct {
		name     string
		images   metal.Images
		machines metal.Machines
		rs       *RethinkStore
		want     metal.Images
		wantErr  bool
	}{
		{
			name:     "simple",
			images:   []metal.Image{ubuntu14_1, ubuntu14_04, ubuntu17_04, ubuntu17_10, ubuntu18_04, ubuntu19_04, ubuntu19_10, alpine3_9, alpine3_10},
			machines: []metal.Machine{testdata.M1, testdata.M9},
			rs:       ds,
			want:     metal.Images{alpine3_9, ubuntu19_04},
			wantErr:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.rs.DeleteOrphanImages(tt.images, tt.machines)
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.DeleteOrphanImages() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RethinkStore.DeleteOrphanImages() = %s", cmp.Diff(got, tt.want))
			}
		})
	}
}

func TestGetOsAndSemver(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		os      string
		version *semver.Version
		wantErr bool
	}{
		{
			name:    "simple",
			id:      "ubuntu-19.04",
			os:      "ubuntu",
			version: semver.MustParse("19.04"),
			wantErr: false,
		},
		{
			name:    "simple2",
			id:      "ubuntu-19.04.20200408",
			os:      "ubuntu",
			version: semver.MustParse("19.04.20200408"),
			wantErr: false,
		},
		{
			name:    "twoparts",
			id:      "ubuntu-small-19.04.20200408",
			os:      "ubuntu-small",
			version: semver.MustParse("19.04.20200408"),
			wantErr: false,
		},
		{
			name:    "fourparts",
			id:      "ubuntu-is-very-small-19.04.20200408",
			os:      "ubuntu-is-very-small",
			version: semver.MustParse("19.04.20200408"),
			wantErr: false,
		},
		{
			name:    "startswithslash",
			id:      "-ubuntu-19.04.20200408",
			os:      "-ubuntu",
			version: semver.MustParse("19.04.20200408"),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os, version, err := GetOsAndSemver(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetOsAndSemver() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if os != tt.os {
				t.Errorf("GetOsAndSemver() got = %v, want %v", os, tt.os)
			}
			if !reflect.DeepEqual(version, tt.version) {
				t.Errorf("GetOsAndSemver() got1 = %v, want %v", os, tt.version)
			}
		})
	}
}
