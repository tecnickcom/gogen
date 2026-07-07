package testutil

import (
	"bytes"
	"errors"
	"io"
)

// ErrorCloser is an [io.ReadCloser] whose Close method always returns an error.
// Its Read side yields io.EOF immediately (an empty body); only Close fails.
// The embedded reader is an [io.Reader] (not a concrete [bytes.Reader]), so the
// exported surface is exactly [io.ReadCloser].
type ErrorCloser struct {
	io.Reader

	err error
}

// NewErrorCloser creates an [ErrorCloser] that returns errMsg from Close.
func NewErrorCloser(errMsg string) *ErrorCloser {
	return &ErrorCloser{
		Reader: bytes.NewReader([]byte{}),
		err:    errors.New(errMsg),
	}
}

// Close always returns the configured error.
func (e *ErrorCloser) Close() error {
	return e.err
}
