package logutil

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSlogWriter_Write(t *testing.T) {
	t.Parallel()

	writer := NewSlogWriter(slog.Default())
	require.NotNil(t, writer)

	t.Run("writes message without newline", func(t *testing.T) {
		t.Parallel()

		msg := "Test log message"
		n, err := writer.Write([]byte(msg))
		require.NoError(t, err)
		require.Equal(t, len(msg), n)
	})

	t.Run("writes message with trailing newline", func(t *testing.T) {
		t.Parallel()

		msg := "Test log message with newline\n"
		n, err := writer.Write([]byte(msg))
		require.NoError(t, err)
		require.Equal(t, len(msg), n)
	})
}

func Test_NewLogFromSlog(t *testing.T) {
	t.Parallel()

	logger := slog.Default()
	stdLogger := NewLogFromSlog(logger)
	require.NotNil(t, stdLogger)

	t.Run("logs message", func(t *testing.T) {
		t.Parallel()

		msg := "Standard log message"
		stdLogger.Print(msg)
	})
}
