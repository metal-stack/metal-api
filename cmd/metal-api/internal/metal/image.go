package metal

import (
	"fmt"
	"strings"
	"time"
)

// An Image describes an image which could be used for provisioning.
type Image struct {
	Base
	URL            string                    `rethinkdb:"url" json:"url"`
	Features       map[ImageFeatureType]bool `rethinkdb:"features" json:"features"`
	OS             string                    `rethinkdb:"os" json:"os"`
	Version        string                    `rethinkdb:"version" json:"version"`
	ExpirationDate time.Time                 `rethinkdb:"expirationDate" json:"expirationDate"`
	// Classification defines the state of a version (preview, supported, deprecated)
	// FIXME implement validation
	Classification VersionClassification `rethinkdb:"classification" json:"classification"`
}

// VersionClassification is the logical state of a version according to
// https://github.wdf.sap.corp/kubernetes/kube-docs/wiki/Versioning-Policy
type VersionClassification string

const (
	// ClassificationPreview indicates that a version has recently been added and not promoted to "Supported" yet.
	// ClassificationPreview versions will not be considered for automatic OperatingSystem patch version updates.
	ClassificationPreview VersionClassification = "preview"
	// ClassificationSupported indicates that a patch version is the default version for the particular minor version.
	// There is always exactly one supported OperatingSystem patch version for every still maintained OperatingSystem minor version.
	// Supported versions are eligible for the automated OperatingSystem patch version update machines.
	ClassificationSupported VersionClassification = "supported"
	// ClassificationDeprecated indicates that a patch version should not be used anymore, should be updated to a new version
	// and will eventually expire.
	// Every version that is neither in preview nor supported is deprecated.
	// All patch versions of not supported minor versions are deprecated.
	ClassificationDeprecated VersionClassification = "deprecated"
)

var versionClassifications = map[string]VersionClassification{
	"preview":    ClassificationPreview,
	"supported":  ClassificationSupported,
	"deprecated": ClassificationDeprecated,
}

// VersionClassificationFrom create a VersionClassification from string
func VersionClassificationFrom(classification string) (VersionClassification, error) {
	vc, ok := versionClassifications[classification]
	if !ok {
		return "", fmt.Errorf("given versionclassification is not valid:%s", classification)
	}
	return vc, nil

}

// ImageFeatureType specifies the features of a images
type ImageFeatureType string

// ImageFeatureString returns the features of an image as a string.
func (i *Image) ImageFeatureString() string {
	features := make([]string, 0, len(i.Features))
	for k := range i.Features {
		features = append(features, string(k))
	}
	return strings.Join(features, ", ")
}

const (
	// ImageFeatureFirewall from this image only a firewall can created
	ImageFeatureFirewall ImageFeatureType = "firewall"
	// ImageFeatureMachine from this image only a machine can created
	ImageFeatureMachine ImageFeatureType = "machine"
)

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
