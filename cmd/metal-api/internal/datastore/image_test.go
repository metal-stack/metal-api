package datastore

import (
	"reflect"
	"testing"
	"time"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/testdata"
	"github.com/Masterminds/semver/v3"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	r "gopkg.in/rethinkdb/rethinkdb-go.v5"
)

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

func Test_getMostRecentImageFor(t *testing.T) {
	i1 := metal.Image{Base: metal.Base{ID: "ubuntu-14.1"}, OS: "ubuntu", Version: "14.1"}
	i2 := metal.Image{Base: metal.Base{ID: "ubuntu-14.04"}, OS: "ubuntu", Version: "14.04"}
	i3 := metal.Image{Base: metal.Base{ID: "ubuntu-17.04"}, OS: "ubuntu", Version: "17.04"}
	i4 := metal.Image{Base: metal.Base{ID: "ubuntu-17.10"}, OS: "ubuntu", Version: "17.10"}
	i5 := metal.Image{Base: metal.Base{ID: "ubuntu-18.04"}, OS: "ubuntu", Version: "18.04"}
	i6 := metal.Image{Base: metal.Base{ID: "ubuntu-19.04"}, OS: "ubuntu", Version: "19.4"}
	i7 := metal.Image{Base: metal.Base{ID: "ubuntu-19.10"}, OS: "ubuntu", Version: "19.10"}
	i8 := metal.Image{Base: metal.Base{ID: "ubuntu-20.04.20200401"}, OS: "ubuntu", Version: "20.04.20200401"}
	i9 := metal.Image{Base: metal.Base{ID: "ubuntu-20.04.20200501"}, OS: "ubuntu", Version: "20.04.20200501"}
	i10 := metal.Image{Base: metal.Base{ID: "ubuntu-20.04.20200502"}, OS: "ubuntu", Version: "20.04.20200502"}
	i11 := metal.Image{Base: metal.Base{ID: "ubuntu-20.04.20200603"}, OS: "ubuntu", Version: "20.04.20200603"}

	i21 := metal.Image{Base: metal.Base{ID: "alpine-3.9"}, OS: "alpine", Version: "3.9"}
	i22 := metal.Image{Base: metal.Base{ID: "alpine-3.9.20191012"}, OS: "alpine", Version: "3.9.20191012"}
	i23 := metal.Image{Base: metal.Base{ID: "alpine-3.10"}, OS: "alpine", Version: "3.10"}
	i24 := metal.Image{Base: metal.Base{ID: "alpine-3.10.20191012"}, OS: "alpine", Version: "3.10.20191012"}
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
			images:  []metal.Image{i10, i7, i3, i8, i6, i1, i9, i5, i2, i4, i11},
			want:    &i6,
			wantErr: false,
		},
		{
			name:    "also simple",
			id:      "ubuntu-19.10",
			images:  []metal.Image{i10, i21, i7, i3, i8, i6, i1, i9, i5, i2, i4, i11, i22},
			want:    &i7,
			wantErr: false,
		},
		{
			name:    "patch given with no match",
			id:      "ubuntu-20.04.2020",
			images:  []metal.Image{i10, i21, i7, i3, i8, i6, i1, i9, i5, i2, i4, i11, i22},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "patch given with match",
			id:      "ubuntu-20.04.20200502",
			images:  []metal.Image{i10, i21, i7, i3, i8, i6, i1, i9, i5, i2, i4, i11, i22},
			want:    &i10,
			wantErr: false,
		},
		{
			name:    "alpine",
			id:      "alpine-3.10",
			images:  []metal.Image{i10, i21, i7, i3, i24, i8, i6, i1, i9, i5, i23, i2, i4, i11, i22},
			want:    &i24,
			wantErr: false,
		},
		{
			name:    "alpine II",
			id:      "alpine-3.9",
			images:  []metal.Image{i10, i21, i7, i3, i24, i8, i6, i1, i9, i5, i23, i2, i4, i11, i22},
			want:    &i22,
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
	i1 := metal.Image{Base: metal.Base{ID: "ubuntu-14.1"}, OS: "ubuntu", Version: "14.1"}
	i2 := metal.Image{Base: metal.Base{ID: "ubuntu-14.04"}, OS: "ubuntu", Version: "14.04"}
	i3 := metal.Image{Base: metal.Base{ID: "ubuntu-17.04"}, OS: "ubuntu", Version: "17.04"}
	i4 := metal.Image{Base: metal.Base{ID: "ubuntu-17.10"}, OS: "ubuntu", Version: "17.10"}
	i5 := metal.Image{Base: metal.Base{ID: "ubuntu-18.04"}, OS: "ubuntu", Version: "18.04"}
	i6 := metal.Image{Base: metal.Base{ID: "ubuntu-19.04"}, OS: "ubuntu", Version: "19.04"}
	i7 := metal.Image{Base: metal.Base{ID: "ubuntu-19.10"}, OS: "ubuntu", Version: "19.10"}
	i8 := metal.Image{Base: metal.Base{ID: "ubuntu-20.04.20200401"}, OS: "ubuntu", Version: "20.04.20200401"}
	i9 := metal.Image{Base: metal.Base{ID: "ubuntu-20.04.20200501"}, OS: "ubuntu", Version: "20.04.20200501"}
	i10 := metal.Image{Base: metal.Base{ID: "ubuntu-20.04.20200502"}, OS: "ubuntu", Version: "20.04.20200502"}
	i11 := metal.Image{Base: metal.Base{ID: "ubuntu-20.04.20200603"}, OS: "ubuntu", Version: "20.04.20200603"}

	i21 := metal.Image{Base: metal.Base{ID: "alpine-3.9"}, OS: "alpine", Version: "3.9"}
	i22 := metal.Image{Base: metal.Base{ID: "alpine-3.10"}, OS: "alpine", Version: "3.10"}

	tests := []struct {
		name   string
		images []metal.Image
		want   []metal.Image
	}{
		{
			name:   "ubuntu versions",
			images: []metal.Image{i10, i7, i3, i8, i6, i1, i9, i5, i2, i4, i11},
			want:   []metal.Image{i11, i10, i9, i8, i7, i6, i5, i4, i3, i2, i1},
		},
		{
			name:   "ubuntu and alpine versions",
			images: []metal.Image{i10, i21, i7, i3, i8, i6, i1, i9, i5, i2, i4, i11, i22},
			want:   []metal.Image{i11, i10, i9, i8, i7, i6, i5, i4, i3, i2, i1, i22, i21},
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

func TestRethinkStore_DeleteOrphanImages(t *testing.T) {
	ds, mock := InitMockDB()
	testdata.InitMockDBData(mock)

	invalid := time.Now().Add(time.Hour * -1)
	valid := time.Now().Add(time.Hour)
	i1 := metal.Image{Base: metal.Base{ID: "ubuntu-14.1"}, OS: "ubuntu", Version: "14.1", ValidTo: valid}
	i2 := metal.Image{Base: metal.Base{ID: "ubuntu-14.04"}, OS: "ubuntu", Version: "14.04", ValidTo: valid}
	i3 := metal.Image{Base: metal.Base{ID: "ubuntu-17.04"}, OS: "ubuntu", Version: "17.04", ValidTo: valid}
	i4 := metal.Image{Base: metal.Base{ID: "ubuntu-17.10"}, OS: "ubuntu", Version: "17.10", ValidTo: valid}
	i5 := metal.Image{Base: metal.Base{ID: "ubuntu-18.04"}, OS: "ubuntu", Version: "18.04", ValidTo: valid}
	i6 := metal.Image{Base: metal.Base{ID: "ubuntu-19.04"}, OS: "ubuntu", Version: "19.04", ValidTo: invalid} // not allocated
	i7 := metal.Image{Base: metal.Base{ID: "ubuntu-19.10"}, OS: "ubuntu", Version: "19.10", ValidTo: invalid} // allocated
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
			images:   []metal.Image{i1, i2, i3, i4, i5, i6, i7},
			machines: []metal.Machine{testdata.M1, testdata.M9},
			rs:       ds,
			want:     metal.Images{i6},
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
