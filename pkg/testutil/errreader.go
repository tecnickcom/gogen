package testutil

import "errors"

// ErrorReader is an io.Reader that always returns an error.
type ErrorReader struct {
	err error
}

// NewErrorReader creates a new errIoReader that always returns the given error message.
func NewErrorReader(msg string) *ErrorReader {
	return &ErrorReader{err: errors.New(msg)}
}

// Read always returns an error.
func (r *ErrorReader) Read([]byte) (int, error) {
	return 0, r.err
}
