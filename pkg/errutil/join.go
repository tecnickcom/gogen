package errutil

import "errors"

// ErrorFunc is a function type that returns an error.
type ErrorFunc func() error

// JoinFnError appends the error returned by fn to the error pointed to by err.
func JoinFnError(err *error, fn ErrorFunc) {
	*err = errors.Join(*err, fn())
}
