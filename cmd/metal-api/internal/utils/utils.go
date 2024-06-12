package utils

import (
	"errors"
	"strconv"
	"strings"

	"github.com/Masterminds/semver/v3"
)

func SplitCIDR(cidr string) (string, *int) {
	parts := strings.Split(cidr, "/")
	if len(parts) == 2 {
		length, err := strconv.Atoi(parts[1])
		if err != nil {
			return parts[0], nil
		}
		return parts[0], &length
	}

	return cidr, nil
}

// GetOsAndSemverFromImage parses a imageID to OS and Semver, or returns an error
// the last part must be the semantic version, valid ids are:
// ubuntu-19.04                 os: ubuntu version: 19.04
// ubuntu-19.04.20200408        os: ubuntu version: 19.04.20200408
// ubuntu-small-19.04.20200408  os: ubuntu-small version: 19.04.20200408
func GetOsAndSemverFromImage(id string) (string, *semver.Version, error) {
	imageParts := strings.Split(id, "-")
	if len(imageParts) < 2 {
		return "", nil, errors.New("image does not contain a version")
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
