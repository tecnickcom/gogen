package logutil

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
)

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
