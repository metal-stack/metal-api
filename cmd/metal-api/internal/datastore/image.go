package datastore

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
)

// GetImage return a image for a given id without semver matching.
func (rs *RethinkStore) GetImage(id string) (*metal.Image, error) {
	var i metal.Image
	err := rs.findEntityByID(rs.imageTable(), &i, id)
	if err != nil {
		return nil, err
	}
	return &i, nil
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

// MigrateMachineImages check images of all machine allocations and migrate them to semver images
// must be executed only once.
func (rs *RethinkStore) MigrateMachineImages(machines metal.Machines) (metal.Machines, error) {
	if machines == nil {
		ms, err := rs.ListMachines()
		if err != nil {
			return nil, err
		}
		machines = ms
	}

	allImages, err := rs.ListImages()
	if err != nil {
		return nil, err
	}

	url2ImageID := make(map[string]string)
	for _, i := range allImages {
		// only consider semver images
		if i.OS == "" || i.Version == "" {
			continue
		}
		url2ImageID[i.URL] = i.ID
	}

	var newMachines metal.Machines
	for _, m := range machines {
		if m.Allocation == nil {
			continue
		}

		imageID := m.Allocation.ImageID
		var i metal.Image
		err = rs.findEntityByID(rs.imageTable(), &i, imageID)
		if err != nil {
			return nil, fmt.Errorf("unable to find image by image ID:%s err:%v", imageID, err)
		}

		semverImageID, ok := url2ImageID[i.URL]
		newMachine := m
		if !ok {
			// no semver image with url configured, use most recent imageID
			// check if given imageID is resolvable to a semver image
			semverImage, err := rs.FindImage(imageID)
			if err != nil {
				return nil, fmt.Errorf("image:%s does not have a matching semver image by url err:%v", imageID, err)
			}
			semverImageID = semverImage.ID
		}
		newMachine.Allocation.ImageID = semverImageID
		err := rs.UpdateMachine(&m, &newMachine)
		if err != nil {
			return nil, err
		}
		newMachines = append(newMachines, newMachine)
	}
	return newMachines, nil
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
	for _, image := range sortedImages {
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
				return nil, fmt.Errorf("unable to delete image:%s err:%v", image.ID, err)
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
	os, sv, err := GetOsAndSemver(id)
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
	for _, image := range sortedImages {
		if !strings.HasPrefix(id, image.OS) {
			continue
		}
		if time.Since(image.ExpirationDate) > 0 {
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
// the last part must be the semantic version, valid ids are:
// ubuntu-19.04                 os: ubuntu version: 19.04
// ubuntu-19.04.20200408        os: ubuntu version: 19.04.20200408
// ubuntu-small-19.04.20200408  os: ubuntu-small version: 19.04.20200408
func GetOsAndSemver(id string) (string, *semver.Version, error) {
	imageParts := strings.Split(id, "-")
	if len(imageParts) < 2 {
		return "", nil, fmt.Errorf("image does not contain a version")
	}

	parts := len(imageParts) - 1
	os := strings.Join(imageParts[:parts], "-")
	version := strings.Join(imageParts[parts:], "")
	v, err := semver.NewVersion(version)
	if err != nil {
		return "", nil, err
	}
	return os, v, nil
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
