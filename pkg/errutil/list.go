package errutil

// Errors returns the individual errors aggregated within err.
//
// It lets callers enumerate the parts of an aggregate produced by [errors.Join]:
//
//   - a nil err yields a nil slice;
//   - an err implementing Unwrap() []error (such as the value returned by
//     [errors.Join]) yields a copy of its aggregated errors;
//   - any other non-nil err yields a single-element slice holding err.
//
// The returned slice never aliases the internal storage of the aggregate error,
// so callers may retain and modify it freely.
func Errors(err error) []error {
	if err == nil {
		return nil
	}

	joined, ok := err.(interface{ Unwrap() []error })
	if !ok {
		return []error{err}
	}

	parts := joined.Unwrap()
	out := make([]error, len(parts))
	copy(out, parts)

	return out
}
