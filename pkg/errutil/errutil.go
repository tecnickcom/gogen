/*
Package errutil provides small, reusable helpers for consistent error handling in
Go applications.

Many Go services repeat the same error-handling patterns: appending cleanup
errors, enriching failures with call-site context, and preserving compatibility
with the standard errors API. This package centralizes those patterns so teams
can keep error paths concise and predictable.

Top features:

  - Trace annotates an error with runtime caller metadata (file, line, function)
    while preserving the original error with %w wrapping. This gives developers
    immediate debugging context without losing errors.Is/errors.As behavior.
  - JoinFnError executes an error-producing function and joins its result into an
    existing error value using errors.Join. This is useful for defer/cleanup logic
    where secondary failures must not overwrite the primary error.
  - Nil-safe behavior: Trace(nil) returns nil, and JoinFnError naturally supports
    nil and non-nil combinations through errors.Join semantics.

Benefits:

  - reduces boilerplate in error-return and defer paths
  - improves observability and diagnosis of production failures
  - encourages idiomatic Go error composition based on errors.Join and %w
*/
package errutil
