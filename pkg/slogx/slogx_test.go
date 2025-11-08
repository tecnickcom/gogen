package slogx

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_NewNop(t *testing.T) {
	t.Parallel()

	logger := NewNop()
	require.NotNil(t, logger)

	logger.Debug("debug message", "key1", "value1")
	logger.Info("info message", "key2", "value2")
	logger.Warn("warn message", "key3", "value3")
	logger.Error("error message", "key4", "value4")

	withLogger := logger.With("key5", "value5")
	require.NotNil(t, withLogger)
}
