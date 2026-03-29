package errutil

import "errors"

// ErrorFunc is a deferred cleanup/action callback that may return an error.
type ErrorFunc func() error

// JoinFnError executes fn and joins its error into *err.
//
// This helper is intended for defer/cleanup paths where secondary failures
// should be preserved without overwriting a primary error.
func JoinFnError(err *error, fn ErrorFunc) {
	*err = errors.Join(*err, fn())
}
