package logutil

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"strings"
	"sync"
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

func TestSlogTraceIDHandler_EmptyWithReturnsReceiver(t *testing.T) {
	t.Parallel()

	base := newSlogTraceIDHandler(slog.DiscardHandler, func() string { return "id" })

	require.Same(t, base, base.WithAttrs(nil), "empty WithAttrs must return the receiver")
	require.Same(t, base, base.WithGroup(""), "empty WithGroup must return the receiver")
}

// TestSlogTraceIDHandler_TraceIDStaysAtRootUnderGroup asserts structural placement:
// even when the logger opens a group, the injected trace ID must remain a root-level
// field (and the grouped attributes must stay inside the group).
func TestSlogTraceIDHandler_TraceIDStaysAtRootUnderGroup(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	cfg, err := NewConfig(
		WithOutWriter(&buf),
		WithFormat(FormatJSON),
		WithTraceIDFn(func() string { return "trace-root" }),
	)
	require.NoError(t, err)

	logger := cfg.SlogLogger().With("base", 1).WithGroup("g")
	logger.Info("msg", "k", "v")

	var got map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))

	require.Equal(t, "trace-root", got[TraceIDKey], "trace_id must be at the root, not nested in the group")

	group, ok := got["g"].(map[string]any)
	require.True(t, ok, "the group must be present")
	require.Equal(t, "v", group["k"], "grouped attributes must stay inside the group")
	require.NotContains(t, group, TraceIDKey, "trace_id must not be nested inside the group")
}

// TestSlogTraceIDHandler_UserTraceIDNotDuplicated verifies that a caller-supplied
// root-level trace_id is not duplicated by the injected one (single key, caller wins).
func TestSlogTraceIDHandler_UserTraceIDNotDuplicated(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	cfg, err := NewConfig(
		WithOutWriter(&buf),
		WithFormat(FormatJSON),
		WithTraceIDFn(func() string { return "CONFIGURED" }),
	)
	require.NoError(t, err)

	// "other" exercises the scan continuing past a non-matching attr before the match.
	cfg.SlogLogger().Info("m", "other", 1, "trace_id", "USER")

	out := buf.String()
	require.Equal(t, 1, strings.Count(out, `"trace_id":`), "exactly one trace_id key")
	require.Contains(t, out, `"trace_id":"USER"`, "the caller-supplied trace_id wins")
}

// TestSlogTraceIDHandler_TraceFnNotCalledWhenDeduped verifies that TraceIDFn is not invoked
// when a caller-supplied trace ID makes the injected one redundant, and is invoked otherwise.
func TestSlogTraceIDHandler_TraceFnNotCalledWhenDeduped(t *testing.T) {
	t.Parallel()

	var (
		buf   bytes.Buffer
		calls atomic.Int64
	)

	cfg, err := NewConfig(
		WithOutWriter(&buf),
		WithFormat(FormatJSON),
		WithTraceIDFn(func() string {
			calls.Add(1)
			return "CFG"
		}),
	)
	require.NoError(t, err)

	l := cfg.SlogLogger()

	l.Info("deduped", "trace_id", "USER") // caller supplies it -> TraceIDFn must be skipped
	require.Equal(t, int64(0), calls.Load(), "TraceIDFn must not run when the trace ID is deduped away")

	l.Info("normal") // no caller trace ID -> TraceIDFn runs
	require.Equal(t, int64(1), calls.Load(), "TraceIDFn must run when it provides the trace ID")
}

// TestSlogTraceIDHandler_ConcurrentGrouped exercises the group-aware slow path from many
// goroutines under -race to confirm it holds no shared mutable state.
func TestSlogTraceIDHandler_ConcurrentGrouped(t *testing.T) {
	t.Parallel()

	cfg, err := NewConfig(
		WithOutWriter(io.Discard),
		WithFormat(FormatJSON),
		WithTraceIDFn(func() string { return "tid" }),
	)
	require.NoError(t, err)

	logger := cfg.SlogLogger().With("k", "v").WithGroup("g") // grouped -> slow path

	var wg sync.WaitGroup

	for range 16 {
		wg.Go(func() {
			for range 50 {
				logger.Info("concurrent", "a", 1)
			}
		})
	}

	wg.Wait()
}

func TestNewSlogTraceIDHandler(t *testing.T) {
	t.Parallel()

	base := slog.DiscardHandler

	// A nil TraceIDFunc returns the handler unchanged (no wrapper).
	got := NewSlogTraceIDHandler(base, nil)
	_, wrapped := got.(*slogTraceIDHandler)
	require.False(t, wrapped, "nil TraceIDFunc must not wrap the handler")

	// A non-nil TraceIDFunc returns a wrapping *slogTraceIDHandler.
	got = NewSlogTraceIDHandler(base, func() string { return "id" })
	_, wrapped = got.(*slogTraceIDHandler)
	require.True(t, wrapped, "non-nil TraceIDFunc must wrap the handler")
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

	require.Contains(t, buf.String(), `"`+TraceIDKey+`":"trace-abc-123"`)
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
