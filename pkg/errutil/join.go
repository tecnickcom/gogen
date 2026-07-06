package errutil

import "errors"

// ErrorFunc is a deferred cleanup/action callback that may return an error.
type ErrorFunc func() error

// ErrNilErrorFunc is joined into the target error by JoinFnError when the
// supplied ErrorFunc is nil, surfacing the programming mistake without panicking.
var ErrNilErrorFunc = errors.New("errutil: nil ErrorFunc")

// JoinFnError executes fn and joins its error into *err.
//
// This helper is intended for defer/cleanup paths where secondary failures
// should be preserved without overwriting a primary error. It is typically used
// as:
//
//	func do() (err error) {
//		f, err := open()
//		if err != nil {
//			return err
//		}
//		defer errutil.JoinFnError(&err, f.Close)
//		// ...
//	}
//
// err must point at the error that will ultimately be returned, usually the
// address of a named return value; passing the address of a local variable that
// is not returned silently discards the joined error.
//
// If err is nil the call is a no-op. If fn is nil, ErrNilErrorFunc is joined
// into *err. The primary error is left untouched when fn returns nil.
func JoinFnError(err *error, fn ErrorFunc) {
	if err == nil {
		return
	}

	if fn == nil {
		*err = errors.Join(*err, ErrNilErrorFunc)

		return
	}

	e := fn()
	if e != nil {
		*err = errors.Join(*err, e)
	}
}
