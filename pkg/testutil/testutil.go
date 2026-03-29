/*
Package testutil solves repetitive test setup problems that appear across Go
codebases: forcing I/O failures on demand, capturing process output,
bootstrapping HTTP handlers, and normalizing time-variant values in assertions.

# Problem

Unit and integration tests often need infrastructure behavior that is hard to
trigger deterministically in production code: `io.Reader` errors, `io.Closer`
errors, router wiring for request/response assertions, and snapshot comparisons
for payloads that contain changing timestamps. Reimplementing those helpers in
every package leads to inconsistent tests and duplicated boilerplate.

testutil centralizes these patterns into small, focused helpers designed for
test-only usage.

# Key Features

  - Deterministic I/O failure mocks:
    [NewErrorReader] returns a reader whose [ErrorReader.Read] always fails;
    [NewErrorCloser] returns an [io.ReadCloser] whose [ErrorCloser.Close]
    always fails. These are useful for exercising error paths that are
    otherwise difficult to trigger.
  - HTTP test bootstrap:
    [RouterWithHandler] builds an [http.Handler] backed by
    `julienschmidt/httprouter` with a route pre-registered, reducing per-test
    router setup noise.
  - Output capture for assertions:
    [CaptureOutput] redirects stdout, stderr, and the default logger for the
    duration of a function call and returns captured output as a string,
    enabling stable assertions for CLI/log-producing code.
  - Time-variant text normalization:
    [ReplaceDateTime] and [ReplaceUnixTimestamp] replace dynamic timestamp
    fragments in strings (for example JSON responses), making golden/snapshot
    assertions deterministic.

# Benefits

  - Less test boilerplate and more consistent test patterns across packages.
  - Easier coverage of failure branches and edge cases.
  - More stable, less flaky string-based assertions.

# Usage

	reader := testutil.NewErrorReader("read failed")
	closer := testutil.NewErrorCloser("close failed")
	_ = reader
	_ = closer

	output := testutil.CaptureOutput(t, func() {
	    fmt.Println("hello")
	})

	h := testutil.RouterWithHandler(http.MethodGet, "/health", func(w http.ResponseWriter, _ *http.Request) {
	    w.WriteHeader(http.StatusOK)
	})
	_ = h

	normalized := testutil.ReplaceDateTime(responseBody, "<DATETIME>")
	normalized = testutil.ReplaceUnixTimestamp(normalized, "<UNIX_TS>")
	_ = output
	_ = normalized

These functions are intended for use in tests only.
*/
package testutil
