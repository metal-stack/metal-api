package datastore

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/utils"
)

// GetImage return a image for a given id without semver matching.
func (rs *RethinkStore) GetImage(ctx context.Context, id string) (*metal.Image, error) {
	var i metal.Image
	err := rs.findEntityByID(ctx, rs.imageTable(), &i, id)
	if err != nil {
		return nil, err
	}
	return &i, nil
}

// FindImages returns all images for the given image id.
func (rs *RethinkStore) FindImages(ctx context.Context, id string) ([]metal.Image, error) {
	allImages, err := rs.ListImages(ctx)
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
func (rs *RethinkStore) FindImage(ctx context.Context, id string) (*metal.Image, error) {
	allImages, err := rs.ListImages(ctx)
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
func (rs *RethinkStore) ListImages(ctx context.Context) (metal.Images, error) {
	imgs := make(metal.Images, 0)
	err := rs.listEntities(ctx, rs.imageTable(), &imgs)
	return imgs, err
}

// CreateImage creates a new image.
func (rs *RethinkStore) CreateImage(ctx context.Context, i *metal.Image) error {
	return rs.createEntity(ctx, rs.imageTable(), i)
}

// DeleteImage deletes an image.
func (rs *RethinkStore) DeleteImage(ctx context.Context, i *metal.Image) error {
	return rs.deleteEntity(ctx, rs.imageTable(), i)
}

// UpdateImage updates an image.
func (rs *RethinkStore) UpdateImage(ctx context.Context, oldImage *metal.Image, newImage *metal.Image) error {
	return rs.updateEntity(ctx, rs.imageTable(), newImage, oldImage)
}

// DeleteOrphanImages deletes Images which are no longer allocated by a machine and older than allowed.
// Always at least one image per OS is kept even if no longer valid and not allocated.
// This ensures to have always at least a usable image left.
func (rs *RethinkStore) DeleteOrphanImages(ctx context.Context, images metal.Images, machines metal.Machines) (metal.Images, error) {
	if images == nil {
		is, err := rs.ListImages(ctx)
		if err != nil {
			return nil, err
		}
		images = is
	}
	if machines == nil {
		ms, err := rs.ListMachines(ctx)
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
			err := rs.DeleteImage(ctx, &image)
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
	os, sv, err := utils.GetOsAndSemverFromImage(id)
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
	os, sv, err := utils.GetOsAndSemverFromImage(id)
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
