package testutil

import (
	"bytes"
	"errors"
	"io"
)

// ErrorCloser is an io.ReadCloser that always returns an error on Close.
type ErrorCloser struct {
	*bytes.Reader

	errMsg string
}

// NewErrorCloser creates a ReadCloser that returns the specified error when closed.
func NewErrorCloser(errMsg string) io.ReadCloser {
	return &ErrorCloser{
		Reader: bytes.NewReader([]byte{}),
		errMsg: errMsg,
	}
}

// Close always returns an error.
func (e *ErrorCloser) Close() error {
	return errors.New(e.errMsg)
}
