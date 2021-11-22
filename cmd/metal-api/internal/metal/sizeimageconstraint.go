package metal

import (
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
)

// SizeImageConstraint expresses optional restrictions for specific size to image combinations
// this might be required if the support for a specific hardware in a given size is only supported
// with a newer version of the image.
//
// If the size in question is not found, no restrictions apply.
// If the image in question is not found, no restrictions apply as well.
// If the image in question is found, but does not match the given expression, machine creation must be forbidden.
type SizeImageConstraint struct {
	Base
	// Images a map from imageID to semver compatible matcher string
	// example:
	// images:
	//    ubuntu: ">= 20.04.20211011"
	//    debian: ">= 10.0.20210101"
	Images map[string]string `rethinkdb:"images" json:"images"`
}

// SizeImageConstraints is a slice of ImageConstraint
type SizeImageConstraints []SizeImageConstraint

func (scs *SizeImageConstraints) Validate() error {
	for _, c := range *scs {
		err := c.Validate()
		if err != nil {
			return err
		}
	}
	return nil
}

func (sc *SizeImageConstraint) Validate() error {
	for os, vc := range sc.Images {
		// no pure wildcard in images
		if os == "*" {
			return fmt.Errorf("just '*' is not allowed as image os constraint")
		}
		// a single "*" is possible
		if strings.TrimSpace(vc) == "*" {
			continue
		}
		_, _, err := convertToOpAndVersion(vc)
		if err != nil {
			return err
		}
	}
	return nil
}

func (scs *SizeImageConstraints) Matches(size Size, image Image) (bool, error) {
	for _, sc := range *scs {
		if sc.ID == size.ID {
			return sc.Matches(size, image)
		}
	}
	return true, nil
}

func (sc *SizeImageConstraint) Matches(size Size, image Image) (bool, error) {
	if sc.ID != size.ID {
		return true, nil
	}
	for os, versionconstraint := range sc.Images {
		if os != image.OS {
			continue
		}
		version, err := semver.NewVersion(image.Version)
		if err != nil {
			return false, fmt.Errorf("version of image is invalid %w", err)
		}

		// FIXME is this a valid assumption
		if version.Patch() == 0 {
			return false, fmt.Errorf("no patch version given")
		}
		c, err := semver.NewConstraint(versionconstraint)
		if err != nil {
			return false, fmt.Errorf("versionconstraint %s is invalid %w", versionconstraint, err)
		}
		if !c.Check(version) {
			return false, fmt.Errorf("given size:%s with image:%s does violate constraints:%s", size.ID, image.ID, c.String())
		}
	}
	return true, nil
}
