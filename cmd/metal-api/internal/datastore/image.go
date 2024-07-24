package datastore

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	metalcommon "github.com/metal-stack/metal-lib/pkg/metal"
	r "gopkg.in/rethinkdb/rethinkdb-go.v6"
)

// ImageSearchQuery can be used to search images.
type ImageSearchQuery struct {
	ID             *string  `json:"id" optional:"true"`
	Name           *string  `json:"name" optional:"true"`
	Features       []string `json:"features" optional:"true"`
	OS             *string  `json:"os" optional:"true"`
	Version        *string  `json:"version" optional:"true"`
	Classification *string  `json:"classification" enum:"preview|supported|deprecated" optional:"true"`
}

// GenerateTerm generates the switch search query term.
func (p *ImageSearchQuery) generateTerm(rs *RethinkStore) *r.Term {
	q := *rs.imageTable()

	if p.ID != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("id").Eq(*p.ID)
		})
	}

	if p.Name != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("name").Eq(*p.Name)
		})
	}

	if p.OS != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("os").Eq(*p.OS)
		})
	}

	if p.Version != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("version").Eq(*p.Version)
		})
	}

	if p.Classification != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("classification").Eq(*p.Classification)
		})
	}

	if len(p.Features) > 0 {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("features").HasFields(p.Features)
		})
	}

	return &q
}

// GetImage return a image for a given id without semver matching.
func (rs *RethinkStore) GetImage(id string) (*metal.Image, error) {
	var i metal.Image
	err := rs.findEntityByID(rs.imageTable(), &i, id)
	if err != nil {
		return nil, err
	}
	return &i, nil
}

// FindImages returns all images for the given image id.
func (rs *RethinkStore) FindImages(id string) ([]metal.Image, error) {
	allImages, err := rs.ListImages()
	if err != nil {
		return nil, err
	}
	imgs, err := getImagesFor(id, allImages)
	if err != nil {
		return nil, metal.NotFound("no images for id:%s found:%v", id, err)
	}

	return imgs, nil
}

// FindImage returns an image for the given image id.
func (rs *RethinkStore) FindImage(id string) (*metal.Image, error) {
	allImages, err := rs.ListImages()
	if err != nil {
		return nil, err
	}
	i, err := rs.getMostRecentImageFor(id, allImages)
	if err != nil {
		return nil, metal.NotFound("no image for id:%s found:%v", id, err)
	}
	if i == nil {
		return nil, metal.NotFound("no image for id:%s found", id)
	}
	return i, nil
}

// ListImages returns all images.
func (rs *RethinkStore) ListImages() (metal.Images, error) {
	imgs := make(metal.Images, 0)
	err := rs.listEntities(rs.imageTable(), &imgs)
	return imgs, err
}

// CreateImage creates a new image.
func (rs *RethinkStore) CreateImage(i *metal.Image) error {
	return rs.createEntity(rs.imageTable(), i)
}

// DeleteImage deletes an image.
func (rs *RethinkStore) DeleteImage(i *metal.Image) error {
	return rs.deleteEntity(rs.imageTable(), i)
}

// UpdateImage updates an image.
func (rs *RethinkStore) UpdateImage(oldImage *metal.Image, newImage *metal.Image) error {
	return rs.updateEntity(rs.imageTable(), newImage, oldImage)
}

// SearchImages searches for images by the given parameters.
func (rs *RethinkStore) SearchImages(q *ImageSearchQuery, images *metal.Images) error {
	return rs.searchEntities(q.generateTerm(rs), images)
}

// DeleteOrphanImages deletes Images which are no longer allocated by a machine and older than allowed.
// Always at least one image per OS is kept even if no longer valid and not allocated.
// This ensures to have always at least a usable image left.
func (rs *RethinkStore) DeleteOrphanImages(images metal.Images, machines metal.Machines) (metal.Images, error) {
	if images == nil {
		is, err := rs.ListImages()
		if err != nil {
			return nil, err
		}
		images = is
	}
	if machines == nil {
		ms, err := rs.ListMachines()
		if err != nil {
			return nil, err
		}
		machines = ms
	}
	firstOSImage := make(map[string]bool)
	result := metal.Images{}
	sortedImages := sortImages(images)
	for i := range sortedImages {
		image := sortedImages[i]
		// Always keep the most recent image for one OS even if no machine uses it is not valid anymore
		// this prevents that there is no image at all left if no new images are pushed.
		_, ok := firstOSImage[image.OS]
		if !ok {
			firstOSImage[image.OS] = true
			continue
		}

		if isOrphanImage(image, machines) {
			err := rs.DeleteImage(&image)
			if err != nil {
				return nil, fmt.Errorf("unable to delete image:%s err:%w", image.ID, err)
			}
			result = append(result, image)
		}
	}
	return result, nil
}

// isOrphanImage check if a image is not allocated and older than allowed.
func isOrphanImage(image metal.Image, machines metal.Machines) bool {
	if time.Since(image.ExpirationDate) < 0 {
		return false
	}
	orphan := true
	for _, m := range machines {
		if m.Allocation == nil {
			continue
		}
		if image.ID == m.Allocation.ImageID {
			return false
		}
	}
	return orphan
}

// getMostRecentImageFor
// the id is in the form of: <name>-<version>
// where name is for example ubuntu or firewall
// version must be a semantic version, see https://semver.org/
// we decided to specify the version in the form of major.minor.patch,
// where patch is in the form of YYYYMMDD
// If version is not fully specified, e.g. ubuntu-19.10 or ubuntu-19.10
// then the most recent ubuntu image (ubuntu-19.10.20200407) is returned
// If patch is specified e.g. ubuntu-20.04.20200502 then this exact image is searched.
func (rs *RethinkStore) getMostRecentImageFor(id string, images metal.Images) (*metal.Image, error) {
	os, sv, err := metalcommon.GetOsAndSemverFromImage(id)
	if err != nil {
		return nil, err
	}

	matcher := "~"
	// if patch is given return a exact match
	if sv.Patch() > 0 {
		matcher = "="
	}
	constraint, err := semver.NewConstraint(matcher + sv.String())
	if err != nil {
		return nil, fmt.Errorf("could not create constraint of image version:%s err:%w", sv, err)
	}

	var latestImage *metal.Image
	sortedImages := sortImages(images)
	for i := range sortedImages {
		image := sortedImages[i]
		if os != image.OS {
			continue
		}
		v, err := semver.NewVersion(image.Version)
		if err != nil {
			continue
		}
		if constraint.Check(v) {
			latestImage = &image
			break
		}
	}
	if latestImage != nil {
		return latestImage, nil
	}
	return nil, fmt.Errorf("no image for os:%s version:%s found", os, sv)
}

// getImagesFor
// the id is in the form of: <name>-<version>
// where name is for example ubuntu or firewall
// version must be a semantic version, see https://semver.org/
// we decided to specify the version in the form of major.minor.patch,
// where patch is in the form of YYYYMMDD
// If version is not fully specified, e.g. ubuntu-19.10 or ubuntu-19.10
// then all ubuntu images (ubuntu-19.10.*) are returned
// If patch is specified e.g. ubuntu-20.04.20200502 then this exact image is searched.
func getImagesFor(id string, images metal.Images) ([]metal.Image, error) {
	os, sv, err := metalcommon.GetOsAndSemverFromImage(id)
	if err != nil {
		return nil, err
	}

	matcher := "~"
	// if patch is given return a exact match
	if sv.Patch() > 0 {
		matcher = "="
	}
	constraint, err := semver.NewConstraint(matcher + sv.String())
	if err != nil {
		return nil, fmt.Errorf("could not create constraint of image version:%s err:%w", sv, err)
	}

	result := []metal.Image{}
	for i := range images {
		image := images[i]
		if os != image.OS {
			continue
		}
		v, err := semver.NewVersion(image.Version)
		if err != nil {
			continue
		}
		if !constraint.Check(v) {
			continue
		}
		result = append(result, image)
	}
	return result, nil
}

func sortImages(images []metal.Image) []metal.Image {
	sort.SliceStable(images, func(i, j int) bool {
		c := strings.Compare(images[i].OS, images[j].OS)
		// OS is equal
		if c == 0 {
			iv, err := semver.NewVersion(images[i].Version)
			if err != nil {
				return false
			}
			jv, err := semver.NewVersion(images[j].Version)
			if err != nil {
				return true
			}
			return iv.GreaterThan(jv)
		}
		return c <= 0
	})
	return images
}
