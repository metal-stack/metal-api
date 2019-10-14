package metal

import (
	"fmt"
	"strings"
	"time"
)

// An Image describes an image which could be used for provisioning.
type Image struct {
	Base
	URL      string                    `rethinkdb:"url"`
	Features map[ImageFeatureType]bool `rethinkdb:"features"`
	OS       string                    `rethinkdb:"os"`
	Version  string                    `rethinkdb:"version"`
	ValidTo  time.Time                 `rethinkdb:"validto"`
}

// ImageFeatureType specifies the features of a images
type ImageFeatureType string

const (
	// ImageFeatureFirewall from this image only a firewall can created
	ImageFeatureFirewall ImageFeatureType = "firewall"
	// ImageFeatureMachine from this image only a machine can created
	ImageFeatureMachine ImageFeatureType = "machine"
)

// ImageFeatureString returns the features of an image as a string.
func (i *Image) ImageFeatureString() string {
	features := make([]string, 0, len(i.Features))
	for k := range i.Features {
		features = append(features, string(k))
	}
	return strings.Join(features, ", ")
}

// Images is a collection of images.
type Images []Image

// ImageMap is an indexed map for images.
type ImageMap map[string]Image

// ByID creates an indexed map from an image collection.
func (ii Images) ByID() ImageMap {
	res := make(ImageMap)
	for i, f := range ii {
		res[f.ID] = ii[i]
	}
	return res
}

// HasFeature returns true if this image has given feature enabled, otherwise false.
func (i *Image) HasFeature(feature ImageFeatureType) bool {
	return i.Features[feature]

}

// ImageFeatureTypeFrom a given name to a ImageFeatureType or error.
func ImageFeatureTypeFrom(name string) (ImageFeatureType, error) {
	switch name {
	case string(ImageFeatureFirewall):
		return ImageFeatureFirewall, nil
	case string(ImageFeatureMachine):
		return ImageFeatureMachine, nil
	default:
		return "", fmt.Errorf("unknown ImageFeatureType:%s", name)
	}
}
