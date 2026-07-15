/*
Package errutil provides helpers for error handling in Go applications.

  - Trace annotates an error with runtime caller metadata (file, line, function)
    while preserving the original error with %w wrapping, so errors.Is and
    errors.As continue to work.
  - JoinFnError executes an error-producing function and joins its result into an
    existing error value using errors.Join, for defer/cleanup logic where
    secondary failures must not overwrite the primary error.
  - Errors enumerates the individual errors aggregated within an errors.Join
    value, returning a single-element slice for a plain error and nil for nil.
  - Trace(nil) returns nil, and JoinFnError supports nil and non-nil combinations
    through errors.Join semantics.
*/
package errutil
