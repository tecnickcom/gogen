package testutil

import (
	"bytes"
	"errors"
	"io"
)

// ErrorCloser is an [io.ReadCloser] whose Close method always returns an error.
type ErrorCloser struct {
	*bytes.Reader

	errMsg string
}

// NewErrorCloser creates an [io.ReadCloser] that returns errMsg from Close.
func NewErrorCloser(errMsg string) io.ReadCloser {
	return &ErrorCloser{
		Reader: bytes.NewReader([]byte{}),
		errMsg: errMsg,
	}
}

// Close always returns an error built from the configured message.
func (e *ErrorCloser) Close() error {
	return errors.New(e.errMsg)
}
