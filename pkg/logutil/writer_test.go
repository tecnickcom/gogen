package logutil

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestSlogWriter_ZeroValue pins that the zero value works as documented: it writes to slog.Default at
// LevelInfo. A struct literal is the natural way to pick a level (SlogWriter{Level: LevelWarning}), and
// leaving Logger unset must not panic on the first write.
func TestSlogWriter_ZeroValue(t *testing.T) { //nolint:paralleltest // mutates the slog default.
	var buf bytes.Buffer

	prev := slog.Default()

	defer slog.SetDefault(prev)

	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: LevelDebug})))

	var w SlogWriter // zero value: no Logger, LevelInfo

	require.NotPanics(t, func() {
		n, err := w.Write([]byte("zero value\n"))
		require.NoError(t, err)
		require.Equal(t, len("zero value\n"), n)
	})

	require.Contains(t, buf.String(), `"msg":"zero value"`, "a nil Logger must fall back to slog.Default")
	require.Contains(t, buf.String(), `"level":"INFO"`, "the zero Level is info")
}

func TestNewSlogWriter_NilLogger(t *testing.T) {
	t.Parallel()

	w := NewSlogWriter(nil)
	require.NotNil(t, w)
	require.NotNil(t, w.Logger, "a nil logger must fall back to slog.Default")
	require.Equal(t, LevelError, w.Level, "NewSlogWriter must default to error level")

	require.NotPanics(t, func() {
		n, err := w.Write([]byte("no panic\n"))
		require.NoError(t, err)
		require.Equal(t, len("no panic\n"), n)
	})
}

func TestNewSlogWriterLevel(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: LevelDebug}))

	w := NewSlogWriterLevel(logger, LevelWarning)
	require.Equal(t, LevelWarning, w.Level)

	msg := "warn line\n"
	n, err := w.Write([]byte(msg))
	require.NoError(t, err)
	require.Equal(t, len(msg), n)
	require.Contains(t, buf.String(), "warn line")
	require.Contains(t, buf.String(), `"level":"WARN"`, "bridged output must use the configured level")
}

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
