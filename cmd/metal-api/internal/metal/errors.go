package metal

import (
	"errors"
	"fmt"
	"runtime"
)

var (
	errNotFound = errors.New("NotFound")
	errConflict = errors.New("Conflict")
	// TODO refactor implementations of fmt.Errorf to metal.Internal() in datastore and service
	errInternal = errors.New("Internal")
)

// NotFound creates a new notfound error with a given error message.
func NotFound(format string, args ...interface{}) error {
	return wrapf(errNotFound, format, args...)
}

// IsNotFound checks if an error is a notfound error.
func IsNotFound(e error) bool {
	return errors.Is(e, errNotFound)
}

// Conflict creates a new conflict error with a given error message.
func Conflict(format string, args ...interface{}) error {
	return wrapf(errConflict, format, args...)
}

// IsConflict checks if an error is a conflict error.
func IsConflict(e error) bool {
	return errors.Is(e, errConflict)
}

// Internal creates a new Internal error with a given error message and the original error.
func Internal(err error, format string, args ...interface{}) error {
	return wrapf(errInternal, wrapf(err, format, args...).Error())
}

// IsInternal checks if an error is a Internal error.
func IsInternal(e error) bool {
	return errors.Is(e, errInternal)
}

// wrapf returns an error annotating err with a stack trace
// at the point Wrapf is called, and the format specifier.
// If err is nil, Wrapf returns nil.
func wrapf(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	err = &withMessage{
		cause: err,
		msg:   fmt.Sprintf(format, args...),
	}
	return &withStack{
		err,
		callers(),
	}
}

type withMessage struct {
	cause error
	msg   string
}

func (w *withMessage) Error() string { return w.msg + ": " + w.cause.Error() }

type withStack struct {
	error
	*stack
}
type stack []uintptr

func callers() *stack {
	const depth = 32
	var pcs [depth]uintptr
	n := runtime.Callers(3, pcs[:])
	var st stack = pcs[0:n]
	return &st
}
