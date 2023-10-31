package datastore

import (
	"reflect"
	"testing"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/google/go-cmp/cmp"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/testdata"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_getMostRecentImageFor(t *testing.T) {
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
	}
	rs := &RethinkStore{}

	for i := range tests {
		tt := tests[i]
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

func Test_getMostRecentImageForFirewall(t *testing.T) {
	valid := time.Now().Add(time.Hour)
	firewall2 := metal.Image{Base: metal.Base{ID: "firewall-2.0.20200331"}, OS: "firewall", Version: "2.0.20200331", ExpirationDate: valid}
	firewallubuntu2 := metal.Image{Base: metal.Base{ID: "firewall-ubuntu-2.0.20200331"}, OS: "firewall-ubuntu", Version: "2.0.20200331", ExpirationDate: valid}
	tests := []struct {
		name    string
		id      string
		images  []metal.Image
		want    *metal.Image
		wantErr bool
	}{
		{
			name:    "reverse",
			id:      "firewall-2",
			images:  []metal.Image{firewallubuntu2, firewall2},
			want:    &firewall2,
			wantErr: false,
		},
		{
			name:    "simple",
			id:      "firewall-ubuntu-2",
			images:  []metal.Image{firewall2, firewallubuntu2},
			want:    &firewallubuntu2,
			wantErr: false,
		},
	}
	rs := &RethinkStore{}

	for i := range tests {
		tt := tests[i]
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

func Test_getImagesFor(t *testing.T) {
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

	alpine3_9 := metal.Image{Base: metal.Base{ID: "alpine-3.9"}, OS: "alpine", Version: "3.9", ExpirationDate: valid}
	alpine3_9_20191012 := metal.Image{Base: metal.Base{ID: "alpine-3.9.20191012"}, OS: "alpine", Version: "3.9.20191012", ExpirationDate: valid}
	alpine3_10 := metal.Image{Base: metal.Base{ID: "alpine-3.10"}, OS: "alpine", Version: "3.10", ExpirationDate: valid}
	alpine3_10_20191012 := metal.Image{Base: metal.Base{ID: "alpine-3.10.20191012"}, OS: "alpine", Version: "3.10.20191012", ExpirationDate: valid}
	tests := []struct {
		name    string
		id      string
		images  []metal.Image
		want    []metal.Image
		wantErr bool
	}{
		{
			name:    "simple",
			id:      "ubuntu-20.04",
			images:  []metal.Image{ubuntu20_04_20200502, ubuntu19_10, ubuntu17_04, ubuntu20_04_20200401, ubuntu19_04, ubuntu14_1, ubuntu20_04_20200501, ubuntu18_04, ubuntu14_04, ubuntu17_10, ubuntu20_04_20200603},
			want:    []metal.Image{ubuntu20_04_20200502, ubuntu20_04_20200401, ubuntu20_04_20200501, ubuntu20_04_20200603},
			wantErr: false,
		},
		{
			name:    "patch given with no match",
			id:      "ubuntu-20.04.2020",
			images:  []metal.Image{ubuntu20_04_20200502, alpine3_9, ubuntu19_10, ubuntu17_04, ubuntu20_04_20200401, ubuntu19_04, ubuntu14_1, ubuntu20_04_20200501, ubuntu18_04, ubuntu14_04, ubuntu17_10, ubuntu20_04_20200603, alpine3_9_20191012},
			want:    []metal.Image{},
			wantErr: false,
		},
		{
			name:    "patch given with match",
			id:      "ubuntu-20.04.20200502",
			images:  []metal.Image{ubuntu20_04_20200502, alpine3_9, ubuntu19_10, ubuntu17_04, ubuntu20_04_20200401, ubuntu19_04, ubuntu14_1, ubuntu20_04_20200501, ubuntu18_04, ubuntu14_04, ubuntu17_10, ubuntu20_04_20200603, alpine3_9_20191012},
			want:    []metal.Image{ubuntu20_04_20200502},
			wantErr: false,
		},
		{
			name:    "alpine",
			id:      "alpine-3.10",
			images:  []metal.Image{ubuntu20_04_20200502, alpine3_9, ubuntu19_10, ubuntu17_04, alpine3_10_20191012, ubuntu20_04_20200401, ubuntu19_04, ubuntu14_1, ubuntu20_04_20200501, ubuntu18_04, alpine3_10, ubuntu14_04, ubuntu17_10, ubuntu20_04_20200603, alpine3_9_20191012},
			want:    []metal.Image{alpine3_10_20191012, alpine3_10},
			wantErr: false,
		},
		{
			name:    "alpine II",
			id:      "alpine-3.9",
			images:  []metal.Image{ubuntu20_04_20200502, alpine3_9, ubuntu19_10, ubuntu17_04, alpine3_10_20191012, ubuntu20_04_20200401, ubuntu19_04, ubuntu14_1, ubuntu20_04_20200501, ubuntu18_04, alpine3_10, ubuntu14_04, ubuntu17_10, ubuntu20_04_20200603, alpine3_9_20191012},
			want:    []metal.Image{alpine3_9, alpine3_9_20191012},
			wantErr: false,
		},
	}

	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			got, err := getImagesFor(tt.id, tt.images)
			if (err != nil) != tt.wantErr {
				t.Errorf("getImagesFor() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getImagesFor() %s\n", cmp.Diff(got, tt.want))
			}
		})
	}
}

func Test_sortImages(t *testing.T) {
	firewall2 := metal.Image{Base: metal.Base{ID: "firewall-2.0.20200331"}, OS: "firewall", Version: "2.0.20200331"}
	firewallubuntu2 := metal.Image{Base: metal.Base{ID: "firewall-ubuntu-2.0.20200331"}, OS: "firewall-ubuntu", Version: "2.0.20200331"}
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
		{
			name:   "firewall",
			images: []metal.Image{firewall2, firewallubuntu2},
			want:   []metal.Image{firewall2, firewallubuntu2},
		},
		{
			name:   "firewall reverse",
			images: []metal.Image{firewallubuntu2, firewall2},
			want:   []metal.Image{firewall2, firewallubuntu2},
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			if got := sortImages(tt.images); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("sortImages() \n%s", cmp.Diff(got, tt.want))
			}
		})
	}
}

func TestSemver(t *testing.T) {
	c, err := semver.NewConstraint("~1")
	require.NoError(t, err)
	assert.NotNil(t, c)
	v, err := semver.NewVersion("1.99.99")
	require.NoError(t, err)
	assert.NotNil(t, v)
	satisfies := c.Check(v)
	assert.True(t, satisfies)
	v, err = semver.StrictNewVersion("19.01")
	require.Error(t, err)
	assert.Nil(t, v)
}

func TestRethinkStore_DeleteOrphanImages(t *testing.T) {
	ds, mock := InitMockDB(t)
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
	for i := range tests {
		tt := tests[i]
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
		{
			name:    "firewall",
			id:      "firewall-2.04.20200408",
			os:      "firewall",
			version: semver.MustParse("2.04.20200408"),
			wantErr: false,
		},
		{
			name:    "firewall-ubuntu",
			id:      "firewall-ubuntu-2.04.20200408",
			os:      "firewall-ubuntu",
			version: semver.MustParse("2.04.20200408"),
			wantErr: false,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			os, version, err := utils.GetOsAndSemverFromImage(tt.id)
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
