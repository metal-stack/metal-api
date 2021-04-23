package metal

import (
	"github.com/pkg/errors"
)

var (
	errNotFound  = errors.New("NotFound")
	errConflict  = errors.New("Conflict")
	errForbidden = errors.New("Forbidden")
	// TODO refactor implementations of fmt.Errorf to metal.Internal() in datastore and service
	errInternal = errors.New("Internal")
)

// NotFound creates a new notfound error with a given error message.
func NotFound(format string, args ...interface{}) error {
	return errors.Wrapf(errNotFound, format, args...)
}

// IsNotFound checks if an error is a notfound error.
func IsNotFound(e error) bool {
	return errors.Is(errors.Cause(e), errNotFound)
}

// Conflict creates a new conflict error with a given error message.
func Conflict(format string, args ...interface{}) error {
	return errors.Wrapf(errConflict, format, args...)
}

// IsConflict checks if an error is a conflict error.
func IsConflict(e error) bool {
	return errors.Is(errors.Cause(e), errConflict)
}

// Internal creates a new Internal error with a given error message and the original error.
func Internal(err error, format string, args ...interface{}) error {
	return errors.Wrap(errInternal, errors.Wrapf(err, format, args...).Error())
}

// IsInternal checks if an error is a Internal error.
func IsInternal(e error) bool {
	return errors.Is(errors.Cause(e), errInternal)
}

// Forbidden creates a new forbidden error with a given error message.
func Forbidden(format string, args ...interface{}) error {
	return errors.Wrapf(errForbidden, format, args...)
}

// IsForbidden checks if an error is a forbidden error.
func IsForbidden(e error) bool {
	return errors.Is(errors.Cause(e), errForbidden)
}
