package datastore

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
	"github.com/Masterminds/semver/v3"
)

// FindImage returns an image for the given image id.
func (rs *RethinkStore) FindImage(id string) (*metal.Image, error) {
	allImages, err := rs.ListImages()
	if err != nil {
		return nil, err
	}
	// FIXME consider exposing semver matching to a separate api endpoint
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

// DeleteOrphanImages deletes Images which are no longer allocated by a machine and older than allowed.
func (rs *RethinkStore) DeleteOrphanImages(images metal.Images, machines metal.Machines) (metal.Images, error) {
	result := metal.Images{}
	for _, image := range images.ByID() {
		if isOrphanImage(image, machines) {
			err := rs.DeleteImage(&image)
			if err != nil {
				return nil, fmt.Errorf("unable to delete image:%s err:%v", image.ID, err)
			}
			result = append(result, image)
		}
	}
	return result, nil
}

// isOrphanImage check if a image is not allocated and older than allowed.
func isOrphanImage(image metal.Image, machines metal.Machines) bool {
	if time.Since(image.ValidTo) < 0 {
		return false
	}
	orphan := true
	for _, m := range machines {
		if m.Allocation == nil {
			continue
		}
		if image.ID == m.Allocation.ImageID {
			orphan = false
			continue
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
// If version is not fully specified, e.g. ubuntu-19.4
// then the most recent ubuntu image (ubuntu-19.4.20191007) is returned
// If patch is specified e.g. ubuntu-20.04.20200502 then this exact image is searched.
func (rs *RethinkStore) getMostRecentImageFor(id string, images metal.Images) (*metal.Image, error) {
	os, sv, err := rs.GetOsAndSemver(id)
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
		return nil, fmt.Errorf("could not create constraint of image version:%s err:%v", sv, err)
	}

	var latestImage *metal.Image
	sortedImages := sortImages(images)
	for _, image := range sortedImages {
		if !strings.HasPrefix(id, image.OS) {
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

// GetOsAndSemver parses a imageID to OS and Semver, or returns an error
func (rs *RethinkStore) GetOsAndSemver(id string) (string, *semver.Version, error) {
	imageParts := strings.Split(id, "-")
	if len(imageParts) < 2 {
		return "", nil, fmt.Errorf("image does not contain a version")
	}

	os := imageParts[0]
	version := strings.Join(imageParts[1:], "")
	v, err := semver.NewVersion(version)
	if err != nil {
		return "", nil, err
	}
	return os, v, nil
}

func sortImages(images []metal.Image) []metal.Image {
	sort.SliceStable(images, func(i, j int) bool {
		c := strings.Compare(images[i].OS, images[j].OS)
		if c <= 0 {
			return true
		}
		return false
	})
	sort.SliceStable(images, func(i, j int) bool {
		iv, err := semver.NewVersion(images[i].Version)
		if err != nil {
			return false
		}
		jv, err := semver.NewVersion(images[j].Version)
		if err != nil {
			return true
		}
		return iv.GreaterThan(jv)
	})
	return images
}
