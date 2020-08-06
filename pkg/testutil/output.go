package testutil

import (
	"bytes"
	"io"
	"log"
	"os"
	"sync"
	"testing"
)

// CaptureOutput hijacks and captures stderr and stdout for testing the given function
func CaptureOutput(t *testing.T, fn func()) string {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Errorf("Unexpected error (os.Pipe): %v", err)
	}
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
		if err != nil {
			t.Errorf("Unexpected error (io.Copy): %v", err)
		}
		out <- buf.String()
	}()
	wg.Wait()

	fn() // call the given function

	if err := writer.Close(); err != nil {
		t.Errorf("Unexpected error (writer.Close): %v", err)
	}
	return <-out
}
