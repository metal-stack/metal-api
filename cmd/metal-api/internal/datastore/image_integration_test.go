//go:build integration
// +build integration

package datastore

import (
	"testing"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"github.com/metal-stack/metal-lib/pkg/pointer"
	"github.com/stretchr/testify/assert"
)

type imageTestable struct{}

func (_ *imageTestable) wipe() error {
	_, err := sharedDS.imageTable().Delete().RunWrite(sharedDS.session)
	return err
}

func (_ *imageTestable) create(s *metal.Image) error { // nolint:unused
	return sharedDS.CreateImage(s)
}

func (_ *imageTestable) delete(id string) error { // nolint:unused
	return sharedDS.DeleteImage(&metal.Image{Base: metal.Base{ID: id}})
}

func (_ *imageTestable) update(old *metal.Image, mutateFn func(s *metal.Image)) error { // nolint:unused
	mod := *old
	if mutateFn != nil {
		mutateFn(&mod)
	}

	return sharedDS.UpdateImage(old, &mod)
}

func (_ *imageTestable) find(id string) (*metal.Image, error) { // nolint:unused
	return sharedDS.GetImage(id)
}

func (_ *imageTestable) list() ([]*metal.Image, error) { // nolint:unused
	res, err := sharedDS.ListImages()
	if err != nil {
		return nil, err
	}

	return derefSlice(res), nil
}

func (_ *imageTestable) search(q *ImageSearchQuery) ([]*metal.Image, error) { // nolint:unused
	var res metal.Images
	err := sharedDS.SearchImages(q, &res)
	if err != nil {
		return nil, err
	}

	return derefSlice(res), nil
}

func TestRethinkStore_FindImage(t *testing.T) {
	tt := &imageTestable{}
	defer func() {
		assert.NoError(t, tt.wipe())
	}()

	tests := []findTest[*metal.Image, *ImageSearchQuery]{
		{
			name: "find",
			id:   "2",

			mock: []*metal.Image{
				{Base: metal.Base{ID: "1"}},
				{Base: metal.Base{ID: "2"}},
				{Base: metal.Base{ID: "3"}},
			},
			want: &metal.Image{
				Base: metal.Base{ID: "2"},
			},
			wantErr: nil,
		},
		{
			name:    "not found",
			id:      "4",
			want:    nil,
			wantErr: metal.NotFound(`no image with id "4" found`),
		},
	}
	for i := range tests {
		tests[i].run(t, tt)
	}
}

func TestRethinkStore_SearchImages(t *testing.T) {
	tt := &imageTestable{}
	defer func() {
		assert.NoError(t, tt.wipe())
	}()

	tests := []searchTest[*metal.Image, *ImageSearchQuery]{
		{
			name: "empty result",
			q: &ImageSearchQuery{
				ID: pointer.Pointer("2"),
			},
			mock: []*metal.Image{
				{Base: metal.Base{ID: "1"}},
			},
			want:    nil,
			wantErr: nil,
		},
		{
			name: "search by id",
			q: &ImageSearchQuery{
				ID: pointer.Pointer("2"),
			},
			mock: []*metal.Image{
				{Base: metal.Base{ID: "1"}},
				{Base: metal.Base{ID: "2"}},
				{Base: metal.Base{ID: "3"}},
			},
			want: []*metal.Image{
				{Base: metal.Base{ID: "2"}},
			},
			wantErr: nil,
		},
		{
			name: "search by name",
			q: &ImageSearchQuery{
				Name: pointer.Pointer("b"),
			},
			mock: []*metal.Image{
				{Base: metal.Base{ID: "1", Name: "a"}},
				{Base: metal.Base{ID: "2", Name: "b"}},
				{Base: metal.Base{ID: "3", Name: "c"}},
			},
			want: []*metal.Image{
				{Base: metal.Base{ID: "2", Name: "b"}},
			},
			wantErr: nil,
		},
		{
			name: "search by feature",
			q: &ImageSearchQuery{
				Features: []string{"firewall"},
			},
			mock: []*metal.Image{
				{Base: metal.Base{ID: "1"}, Features: map[metal.ImageFeatureType]bool{"machine": true}},
				{Base: metal.Base{ID: "2"}, Features: map[metal.ImageFeatureType]bool{"firewall": true}},
				{Base: metal.Base{ID: "3"}, Features: map[metal.ImageFeatureType]bool{"firewall": true}},
			},
			want: []*metal.Image{
				{Base: metal.Base{ID: "2"}, Features: map[metal.ImageFeatureType]bool{"firewall": true}},
				{Base: metal.Base{ID: "3"}, Features: map[metal.ImageFeatureType]bool{"firewall": true}},
			},
			wantErr: nil,
		},
		{
			name: "search by multiple features",
			q: &ImageSearchQuery{
				Features: []string{"machine", "firewall"},
			},
			mock: []*metal.Image{
				{Base: metal.Base{ID: "1"}, Features: map[metal.ImageFeatureType]bool{"machine": true}},
				{Base: metal.Base{ID: "2"}, Features: map[metal.ImageFeatureType]bool{"machine": true, "firewall": true}},
				{Base: metal.Base{ID: "3"}, Features: map[metal.ImageFeatureType]bool{"firewall": true}},
			},
			want: []*metal.Image{
				{Base: metal.Base{ID: "2"}, Features: map[metal.ImageFeatureType]bool{"machine": true, "firewall": true}},
			},
			wantErr: nil,
		},
		{
			name: "search by os",
			q: &ImageSearchQuery{
				OS: pointer.Pointer("ubuntu"),
			},
			mock: []*metal.Image{
				{Base: metal.Base{ID: "1"}, OS: "debian"},
				{Base: metal.Base{ID: "2"}, OS: "ubuntu"},
				{Base: metal.Base{ID: "3"}, OS: "debian"},
			},
			want: []*metal.Image{
				{Base: metal.Base{ID: "2"}, OS: "ubuntu"},
			},
			wantErr: nil,
		},
		{
			name: "search by version",
			q: &ImageSearchQuery{
				Version: pointer.Pointer("v2"),
			},
			mock: []*metal.Image{
				{Base: metal.Base{ID: "1"}, Version: "v2"},
				{Base: metal.Base{ID: "2"}, Version: "v1"},
				{Base: metal.Base{ID: "3"}, Version: "v2"},
			},
			want: []*metal.Image{
				{Base: metal.Base{ID: "1"}, Version: "v2"},
				{Base: metal.Base{ID: "3"}, Version: "v2"},
			},
			wantErr: nil,
		},
		{
			name: "search by classification",
			q: &ImageSearchQuery{
				Classification: pointer.Pointer(string(metal.ClassificationPreview)),
			},
			mock: []*metal.Image{
				{Base: metal.Base{ID: "1"}, Classification: metal.ClassificationPreview},
				{Base: metal.Base{ID: "2"}, Classification: metal.ClassificationSupported},
				{Base: metal.Base{ID: "3"}, Classification: metal.ClassificationPreview},
				{Base: metal.Base{ID: "4"}, Classification: metal.ClassificationDeprecated},
			},
			want: []*metal.Image{
				{Base: metal.Base{ID: "1"}, Classification: metal.ClassificationPreview},
				{Base: metal.Base{ID: "3"}, Classification: metal.ClassificationPreview},
			},
			wantErr: nil,
		},
	}

	for i := range tests {
		tests[i].run(t, tt)
	}
}

func TestRethinkStore_ListImages(t *testing.T) {
	tt := &imageTestable{}
	defer func() {
		assert.NoError(t, tt.wipe())
	}()

	tests := []listTest[*metal.Image, *ImageSearchQuery]{
		{
			name: "list",
			mock: []*metal.Image{
				{Base: metal.Base{ID: "1"}},
				{Base: metal.Base{ID: "2"}},
				{Base: metal.Base{ID: "3"}},
			},
			want: []*metal.Image{
				{Base: metal.Base{ID: "1"}},
				{Base: metal.Base{ID: "2"}},
				{Base: metal.Base{ID: "3"}},
			},
		},
	}
	for i := range tests {
		tests[i].run(t, tt)
	}
}

func TestRethinkStore_CreateImage(t *testing.T) {
	tt := &imageTestable{}
	defer func() {
		assert.NoError(t, tt.wipe())
	}()

	tests := []createTest[*metal.Image, *ImageSearchQuery]{
		{
			name: "create",
			want: &metal.Image{
				Base: metal.Base{ID: "1"},
			},
			wantErr: nil,
		},
		{
			name: "already exists",
			mock: []*metal.Image{
				{Base: metal.Base{ID: "1"}},
			},
			want: &metal.Image{
				Base: metal.Base{ID: "1"},
			},
			wantErr: metal.Conflict(`cannot create image in database, entity already exists: 1`),
		},
	}
	for i := range tests {
		tests[i].run(t, tt)
	}
}

func TestRethinkStore_DeleteImage(t *testing.T) {
	tt := &imageTestable{}
	defer func() {
		assert.NoError(t, tt.wipe())
	}()

	tests := []deleteTest[*metal.Image, *ImageSearchQuery]{
		{
			name: "delete",
			id:   "2",
			mock: []*metal.Image{
				{Base: metal.Base{ID: "1"}},
				{Base: metal.Base{ID: "2"}},
				{Base: metal.Base{ID: "3"}},
			},
			want: []*metal.Image{
				{Base: metal.Base{ID: "1"}},
				{Base: metal.Base{ID: "3"}},
			},
		},
		{
			name: "not exists results in noop",
			id:   "abc",
			mock: []*metal.Image{
				{Base: metal.Base{ID: "1"}},
				{Base: metal.Base{ID: "2"}},
				{Base: metal.Base{ID: "3"}},
			},
			want: []*metal.Image{
				{Base: metal.Base{ID: "1"}},
				{Base: metal.Base{ID: "2"}},
				{Base: metal.Base{ID: "3"}},
			},
		},
	}
	for i := range tests {
		tests[i].run(t, tt)
	}
}

func TestRethinkStore_UpdateImage(t *testing.T) {
	tt := &imageTestable{}
	defer func() {
		assert.NoError(t, tt.wipe())
	}()

	tests := []updateTest[*metal.Image, *ImageSearchQuery]{
		{
			name: "update",
			mock: []*metal.Image{
				{Base: metal.Base{ID: "1"}},
				{Base: metal.Base{ID: "2"}},
				{Base: metal.Base{ID: "3"}},
			},
			mutateFn: func(s *metal.Image) {
				s.URL = "url"
			},
			want: &metal.Image{
				Base: metal.Base{ID: "1"},
				URL:  "url",
			},
		},
	}
	for i := range tests {
		tests[i].run(t, tt)
	}
}
