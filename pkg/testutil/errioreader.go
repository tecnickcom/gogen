package testutil

import "errors"

// ErrorIoReader is an io.Reader that always returns an error.
type ErrorIoReader struct {
	err error
}

// NewErrorIoReader creates a new errIoReader that always returns the given error message.
func NewErrorIoReader(msg string) *ErrorIoReader {
	return &ErrorIoReader{err: errors.New(msg)}
}

// Read always returns an error.
func (r *ErrorIoReader) Read([]byte) (int, error) {
	return 0, r.err
}
