package metal

import (
	"fmt"

	"github.com/pkg/errors"
)

var (
	errNotFound = fmt.Errorf("NotFound")
	errConflict = fmt.Errorf("Conflict")
)

// NotFound creates a new notfound error with a given error message.
func NotFound(format string, args ...interface{}) error {
	return errors.Wrapf(errNotFound, format, args...)
}

// IsNotFound checks if an error is a notfound error.
func IsNotFound(e error) bool {
	return errors.Cause(e) == errNotFound
}

// Conflict creates a new conflict error with a given error message.
func Conflict(format string, args ...interface{}) error {
	return errors.Wrapf(errConflict, format, args...)
}

// IsConflict checks if an error is a conflict error.
func IsConflict(e error) bool {
	return errors.Cause(e) == errConflict
}

// ErrorResponse is returned in case of functional errors.
type ErrorResponse struct {
	StatusCode int    `json:"statuscode" description:"http status code"`
	Message    string `json:"message" description:"error message"`
	Operation  string `json:"operation" description:"name of the operation which caused this error"`
}
