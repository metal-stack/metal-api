package utils

import (
	"errors"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/emicklei/go-restful/v3"
	"github.com/metal-stack/metal-lib/zapup"
	"go.uber.org/zap"
)

// Logger returns the request logger from the request.
func Logger(rq *restful.Request) *zap.Logger {
	return zapup.RequestLogger(rq.Request)
}

// CurrentFuncName returns the name of the caller of this function.
func CurrentFuncName() string {
	pc, _, _, ok := runtime.Caller(1)
	if !ok {
		return "unknown"
	}
	ffpc := runtime.FuncForPC(pc)
	if ffpc == nil {
		return "unknown"
	}
	pp := strings.Split(ffpc.Name(), ".")
	return pp[len(pp)-1]
}

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

func StrValueDefault(ptr *string, fallback string) string {
	if ptr == nil {
		return fallback
	}
	return *ptr
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

func UniqueSorted(i []string) []string {
	set := make(map[string]bool)
	for _, e := range i {
		set[e] = true
	}
	unique := []string{}
	for k := range set {
		unique = append(unique, k)
	}
	sort.Strings(unique)
	return unique
}
