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

// CaptureOutput hijacks and captures stderr and stdout for testing the given function.
func CaptureOutput(t *testing.T, fn func()) string {
	t.Helper()

	reader, writer, err := os.Pipe()
	require.Nil(t, err, "Unexpected error (os.Pipe)")

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

	out := make(chan string)
	wg := new(sync.WaitGroup)
	wg.Add(1)

	go func() {
		var buf bytes.Buffer

		wg.Done()

		_, err := io.Copy(&buf, reader)
		require.Nil(t, err, "Unexpected error (io.Copy)")
		out <- buf.String()
	}()

	wg.Wait()

	fn() // call the given function

	err = writer.Close()
	require.Nil(t, err, "Unexpected error (writer.Close)")

	return <-out
}
