package testutil

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestErrorIoReader_Read(t *testing.T) {
	t.Parallel()

	errMsg := "test error"
	reader := NewErrorIoReader(errMsg)

	require.NotNil(t, reader)
	require.Error(t, reader.err)
	require.Equal(t, errMsg, reader.err.Error())

	buf := make([]byte, 1)
	n, err := reader.Read(buf)

	require.Equal(t, 0, n)
	require.Error(t, err)
	require.Equal(t, errMsg, err.Error())
}
