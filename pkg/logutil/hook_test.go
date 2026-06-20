package logutil

import (
	"bytes"
	"log/slog"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewSlogHookHandler(t *testing.T) {
	t.Parallel()

	hkfn := func(_ LogLevel, _ string) {}

	h := NewSlogHookHandler(slog.DiscardHandler, hkfn)

	require.NotNil(t, h)
	require.NotNil(t, h.hookFn)
}

func TestSlogHookHandler_DerivedLoggerFiresHook(t *testing.T) {
	t.Parallel()

	var calls atomic.Int64

	hkfn := func(level LogLevel, message string) {
		require.Equal(t, slog.LevelInfo, level)
		require.Equal(t, "derived", message)

		calls.Add(1)
	}

	var buf bytes.Buffer

	base := NewSlogHookHandler(slog.NewJSONHandler(&buf, nil), hkfn)

	// Derive via both WithAttrs (logger.With) and WithGroup.
	derived := slog.New(base).With("k", "v").WithGroup("g")

	derived.Info("derived")

	require.Equal(t, int64(1), calls.Load(), "hook must still fire for derived loggers")
	require.Contains(t, buf.String(), "derived")
}

func TestSlogHookHandler_WithAttrs(t *testing.T) {
	t.Parallel()

	hkfn := func(_ LogLevel, _ string) {}

	base := NewSlogHookHandler(slog.DiscardHandler, hkfn)

	got := base.WithAttrs([]slog.Attr{slog.String("a", "b")})

	hooked, ok := got.(*SlogHookHandler)
	require.True(t, ok, "WithAttrs must return a *SlogHookHandler that preserves the hook")
	require.NotNil(t, hooked.hookFn)
}

func TestSlogHookHandler_WithGroup(t *testing.T) {
	t.Parallel()

	hkfn := func(_ LogLevel, _ string) {}

	base := NewSlogHookHandler(slog.DiscardHandler, hkfn)

	got := base.WithGroup("g")

	hooked, ok := got.(*SlogHookHandler)
	require.True(t, ok, "WithGroup must return a *SlogHookHandler that preserves the hook")
	require.NotNil(t, hooked.hookFn)
}

func TestSlogTraceIDHandler_TraceIDAppearsInOutput(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	cfg, err := NewConfig(
		WithOutWriter(&buf),
		WithFormat(FormatJSON),
		WithTraceIDFn(func() string { return "trace-abc-123" }),
	)
	require.NoError(t, err)

	logger := cfg.SlogLogger()
	logger.Info("with trace")

	require.Contains(t, buf.String(), `"`+traceIDKey+`":"trace-abc-123"`)
}

func TestSlogTraceIDHandler_DerivedLoggerKeepsTraceID(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	cfg, err := NewConfig(
		WithOutWriter(&buf),
		WithFormat(FormatJSON),
		WithTraceIDFn(func() string { return "trace-xyz" }),
	)
	require.NoError(t, err)

	logger := cfg.SlogLogger().With("k", "v").WithGroup("g")
	logger.Info("derived trace")

	require.Contains(t, buf.String(), "trace-xyz")
}

func TestSlogTraceIDHandler_WithAttrs(t *testing.T) {
	t.Parallel()

	base := newSlogTraceIDHandler(slog.DiscardHandler, func() string { return "id" })

	got := base.WithAttrs([]slog.Attr{slog.String("a", "b")})

	traced, ok := got.(*slogTraceIDHandler)
	require.True(t, ok, "WithAttrs must return a *slogTraceIDHandler that preserves the trace ID func")
	require.NotNil(t, traced.traceIDFn)
}

func TestSlogTraceIDHandler_WithGroup(t *testing.T) {
	t.Parallel()

	base := newSlogTraceIDHandler(slog.DiscardHandler, func() string { return "id" })

	got := base.WithGroup("g")

	traced, ok := got.(*slogTraceIDHandler)
	require.True(t, ok, "WithGroup must return a *slogTraceIDHandler that preserves the trace ID func")
	require.NotNil(t, traced.traceIDFn)
}
