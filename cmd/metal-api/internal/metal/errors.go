package metal

import (
	"errors"
	"fmt"
)

var (
	errNotFound = errors.New("NotFound")
	errConflict = errors.New("Conflict")
	// TODO refactor implementations of fmt.Errorf to metal.Internal() in datastore and service
	errInternal = errors.New("Internal")
)

// NotFound creates a new notfound error with a given error message.
func NotFound(format string, args ...interface{}) error {
	return fmt.Errorf("%w %s", errNotFound, fmt.Sprintf(format, args...))
}

// IsNotFound checks if an error is a notfound error.
func IsNotFound(e error) bool {
	return errors.Is(e, errNotFound)
}

// Conflict creates a new conflict error with a given error message.
func Conflict(format string, args ...interface{}) error {
	return fmt.Errorf("%w %s", errConflict, fmt.Sprintf(format, args...))
}

// IsConflict checks if an error is a conflict error.
func IsConflict(e error) bool {
	return errors.Is(e, errConflict)
}

// Internal creates a new Internal error with a given error message and the original error.
func Internal(err error, format string, args ...interface{}) error {
	return fmt.Errorf("%w %w", errInternal, fmt.Errorf("%w %s", err, fmt.Sprintf(format, args...)))
}

// IsInternal checks if an error is a Internal error.
func IsInternal(e error) bool {
	return errors.Is(e, errInternal)
}
