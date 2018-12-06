package metal

import (
	"fmt"

	"github.com/pkg/errors"
)

var (
	errNotFound = fmt.Errorf("NotFound")
)

// NotFound creates a new notfound error with a given error message.
func NotFound(format string, args ...interface{}) error {
	return errors.Wrapf(errNotFound, format, args...)
}

// IsNotFound checks if an error is a notfound error.
func IsNotFound(e error) bool {
	return errors.Cause(e) == errNotFound
}
