package testutil

import (
	"bytes"
	"io"
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// CaptureOutput captures stdout, stderr, and standard-logger output produced while fn
// runs, and returns it as a single string.
//
// It temporarily redirects the process-global os.Stdout, os.Stderr, and the standard
// logger (via log.SetOutput) to an in-memory pipe, runs fn, then restores all three —
// including the logger's previous output destination — before returning. Restoration
// and cleanup happen on every exit path, including a panic inside fn.
//
// Because it mutates process-global state, CaptureOutput MUST NOT be used from parallel
// tests (t.Parallel) or while other goroutines write to stdout, stderr, or the standard
// logger; concurrent use would capture interleaved or missing output.
func CaptureOutput(t *testing.T, fn func()) string {
	t.Helper()

	reader, writer, err := os.Pipe()
	require.NoError(t, err, "Unexpected error (os.Pipe)")

	origStdout := os.Stdout
	origStderr := os.Stderr
	origLogWriter := log.Writer()

	// Restore global state and release the pipe on every exit path, including a panic in
	// fn or an early require failure. Closing an already-closed pipe end is a harmless
	// no-op, so the explicit writer.Close below and this deferred close cooperate safely.
	defer func() {
		os.Stdout = origStdout
		os.Stderr = origStderr

		log.SetOutput(origLogWriter)

		_ = writer.Close()
		_ = reader.Close()
	}()

	os.Stdout = writer
	os.Stderr = writer
	log.SetOutput(writer)

	type captureResult struct {
		out string
		err error
	}

	// Drain the read end concurrently so fn never blocks on a full pipe buffer. The
	// buffered channel lets the goroutine finish and exit even if we stop receiving early
	// (for example when an intermediate require call fails).
	out := make(chan captureResult, 1)

	go func() {
		var buf bytes.Buffer

		_, copyErr := io.Copy(&buf, reader)
		out <- captureResult{out: buf.String(), err: copyErr}
	}()

	fn() // call the given function

	// Close the write end so the copy goroutine observes EOF and returns.
	err = writer.Close()
	require.NoError(t, err, "Unexpected error (writer.Close)")

	res := <-out
	require.NoError(t, res.err, "Unexpected error (io.Copy)")

	return res.out
}
