package logutil

import (
	"bytes"
	"log/slog"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestNewSlogHookHandler_NilHandler pins the nil-handler guard: the constructor falls back to the
// default logger's handler rather than returning one that panics on first use.
func TestNewSlogHookHandler_NilHandler(t *testing.T) { //nolint:paralleltest // mutates the slog default.
	var (
		buf    bytes.Buffer
		hooked atomic.Int64
	)

	prev := slog.Default()

	defer slog.SetDefault(prev)

	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))

	h := NewSlogHookHandler(nil, func(_ LogLevel, _ string) { hooked.Add(1) })

	require.NotPanics(t, func() { slog.New(h).Info("m") })
	require.Contains(t, buf.String(), `"msg":"m"`, "a nil handler must fall back to the default one")
	require.Equal(t, int64(1), hooked.Load(), "and the hook must still fire")
}

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

func TestSlogHookHandler_NilHookDoesNotPanic(t *testing.T) {
	t.Parallel()

	h := NewSlogHookHandler(slog.DiscardHandler, nil)

	require.NotPanics(t, func() {
		err := h.Handle(t.Context(), slog.Record{Level: slog.LevelInfo, Message: "x"})
		require.NoError(t, err)
	})
}

func TestSlogHookHandler_EmptyWithReturnsReceiver(t *testing.T) {
	t.Parallel()

	base := NewSlogHookHandler(slog.DiscardHandler, func(_ LogLevel, _ string) {})

	require.NotNil(t, base.WithAttrs(nil), "empty WithAttrs must still return a usable handler")
	require.NotNil(t, base.WithGroup(""), "empty WithGroup must still return a usable handler")
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
