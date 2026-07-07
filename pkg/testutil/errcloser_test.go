package testutil

import (
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestErrorCloser_Close(t *testing.T) {
	t.Parallel()

	errMsg := "test error"
	closer := NewErrorCloser(errMsg)

	// Read yields EOF immediately (empty body); only Close fails.
	buf := make([]byte, 8)
	n, readErr := closer.Read(buf)
	require.Equal(t, 0, n)
	require.ErrorIs(t, readErr, io.EOF)

	err := closer.Close()
	require.Error(t, err)
	require.EqualError(t, err, errMsg)
}
