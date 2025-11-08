package testutil

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestErrorCloser_Close(t *testing.T) {
	t.Parallel()

	errMsg := "test error"
	reader := NewErrorCloser(errMsg)

	err := reader.Close()
	require.Error(t, err)
	require.EqualError(t, err, errMsg)
}
