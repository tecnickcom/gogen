package testutil

import "errors"

// ErrorReader is an io.Reader that always returns an error.
type ErrorReader struct {
	err error
}

// NewErrorReader creates an [ErrorReader] that always returns msg as an error.
func NewErrorReader(msg string) *ErrorReader {
	return &ErrorReader{err: errors.New(msg)}
}

// Read always returns the configured error.
func (r *ErrorReader) Read([]byte) (int, error) {
	return 0, r.err
}
