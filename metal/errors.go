package metal

import (
	"fmt"

	"github.com/pkg/errors"
)

var (
	errNotFound = fmt.Errorf("NotFound")
)

func NotFound(format string, args ...interface{}) error {
	return errors.Wrapf(errNotFound, format, args...)
}

func IsNotFound(e error) bool {
	return errors.Cause(e) == errNotFound
}
