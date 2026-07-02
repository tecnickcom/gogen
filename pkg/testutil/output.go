package testutil

import (
	"bytes"
	"io"
	"log"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// CaptureOutput captures stdout, stderr, and default logger output while fn runs.
func CaptureOutput(t *testing.T, fn func()) string {
	t.Helper()

	reader, writer, err := os.Pipe()
	require.NoError(t, err, "Unexpected error (os.Pipe)")

	stdout := os.Stdout
	stderr := os.Stderr

	defer func() {
		os.Stdout = stdout
		os.Stderr = stderr
		log.SetOutput(os.Stderr)
	}()

	os.Stdout = writer
	os.Stderr = writer
	log.SetOutput(writer)

	type captureResult struct {
		out string
		err error
	}

	out := make(chan captureResult, 1)
	wg := new(sync.WaitGroup)
	wg.Add(1)

	go func() {
		var buf bytes.Buffer

		wg.Done()

		// Always send a result so the receive below can never block forever,
		// and surface any pipe read error instead of swallowing it.
		_, err := io.Copy(&buf, reader)
		out <- captureResult{out: buf.String(), err: err}
	}()

	wg.Wait()

	fn() // call the given function

	err = writer.Close()
	require.NoError(t, err, "Unexpected error (writer.Close)")

	res := <-out

	err = reader.Close()
	require.NoError(t, err, "Unexpected error (reader.Close)")

	require.NoError(t, res.err, "Unexpected error (io.Copy)")

	return res.out
}
