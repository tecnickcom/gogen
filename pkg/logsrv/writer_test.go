package logsrv

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"github.com/tecnickcom/nurago/pkg/logutil"
)

func Test_isTerminalWriter(t *testing.T) {
	t.Parallel()

	require.False(t, isTerminalWriter(&bytes.Buffer{}), "a non-file writer is not a terminal")

	f, err := os.CreateTemp(t.TempDir(), "logsrv")
	require.NoError(t, err)

	defer func() { _ = f.Close() }()

	require.False(t, isTerminalWriter(f), "a regular file is not a terminal")
}

// TestNewLogger_severeLevelsDoNotTerminate locks in the invariant that the most severe levels
// emit ordinary records carrying their full syslog name and never terminate the process (the
// handler uses zerolog's NoLevel, so no Panic/Fatal event behavior is ever triggered).
func TestNewLogger_severeLevelsDoNotTerminate(t *testing.T) {
	t.Parallel()

	out := &bytes.Buffer{}

	cfg, err := logutil.NewConfig(
		logutil.WithOutWriter(out),
		logutil.WithFormat(logutil.FormatJSON),
		logutil.WithLevel(logutil.LevelTrace),
	)
	require.NoError(t, err)

	l := NewLogger(cfg)

	require.NotPanics(t, func() {
		l.Log(t.Context(), logutil.LevelEmergency, "emergency msg")
		l.Log(t.Context(), logutil.LevelAlert, "alert msg")
	})

	require.Contains(t, out.String(), `"level":"emergency"`)
	require.Contains(t, out.String(), `"level":"alert"`)
	require.Contains(t, out.String(), "emergency msg")
	require.Contains(t, out.String(), "alert msg")
}

// TestNewHandler exercises the pure constructor: it builds a working handler
// without installing a process-wide default logger.
func TestNewHandler(t *testing.T) {
	t.Parallel()

	out := &bytes.Buffer{}

	cfg, err := logutil.NewConfig(
		logutil.WithOutWriter(out),
		logutil.WithFormat(logutil.FormatJSON),
		logutil.WithLevel(logutil.LevelDebug),
	)
	require.NoError(t, err)

	h := NewHandler(cfg)
	require.NotNil(t, h)

	slog.New(h).Info("via handler")

	require.Contains(t, out.String(), "via handler")
}

// TestNewLogger_nilConfig verifies a nil cfg falls back to logutil.DefaultConfig
// instead of panicking.
func TestNewLogger_nilConfig(t *testing.T) {
	t.Parallel()

	require.NotPanics(t, func() {
		l := NewLogger(nil)
		require.NotNil(t, l)
	})
}

// TestNewHandler_nilOut verifies an unusable Out writer falls back to os.Stderr instead of building a
// handler that panics on first write — including a typed nil, which WithOutWriter rejects but which
// the exported Config.Out field lets a caller assign directly, and which is not caught by a plain
// nil check: the interface holding it is not nil.
func TestNewHandler_nilOut(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		out  io.Writer
	}{
		{name: "untyped nil", out: nil},
		{name: "typed nil", out: (*os.File)(nil)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg, err := logutil.NewConfig(logutil.WithFormat(logutil.FormatJSON))
			require.NoError(t, err)

			cfg.Out = tt.out // hand-assigned writer must fall back, not panic

			var h slog.Handler

			require.NotPanics(t, func() { h = NewHandler(cfg) })
			require.NotNil(t, h)

			// The first write is where an unusable destination would panic: os.Stderr is the fallback,
			// so the record is written there rather than blowing up.
			require.NotPanics(t, func() {
				_ = h.Handle(t.Context(), slog.NewRecord(time.Now(), logutil.LevelInfo, "fallback to stderr", 0))
			}, "the fallback destination must accept the first write")
		})
	}
}

func Test_writerByFormat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		format logutil.LogFormat
		out    io.Writer
		want   io.Writer
	}{
		{
			name:   "json",
			format: logutil.FormatJSON,
			out:    os.Stdout,
			want:   os.Stdout,
		},
		{
			name:   "none",
			format: logutil.FormatNone,
			out:    os.Stdout,
			want:   io.Discard,
		},
		{
			name:   "default",
			format: 56,
			out:    os.Stdout,
			want:   os.Stdout,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := writerByFormat(tt.format, tt.out)

			require.Equal(t, tt.want, got)
		})
	}
}

// failingWriter fails every write, like a full disk or a closed pipe.
type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errWriteFailed }

var errWriteFailed = errors.New("disk on fire")

// TestNewHandler_WriteErrorReported pins that a failed write is reported to the caller rather than
// silently swallowed: zerolog's Event API returns no error of its own, so the destination is wrapped
// to remember one. A subsequent successful write reports nothing, so each failure surfaces once.
func TestNewHandler_WriteErrorReported(t *testing.T) {
	t.Parallel()

	cfg, err := logutil.NewConfig(
		logutil.WithOutWriter(failingWriter{}),
		logutil.WithFormat(logutil.FormatJSON),
		logutil.WithLevel(logutil.LevelDebug),
	)
	require.NoError(t, err)

	h := NewHandler(cfg)
	rec := slog.NewRecord(time.Now(), logutil.LevelInfo, "m", 0)

	require.ErrorIs(t, h.Handle(t.Context(), rec), errWriteFailed, "a failed write must be reported")

	// A grouped handler reports it too.
	require.ErrorIs(t, h.WithGroup("g").Handle(t.Context(), rec), errWriteFailed)

	// A working destination reports nothing.
	ok, err := logutil.NewConfig(logutil.WithOutWriter(io.Discard), logutil.WithLevel(logutil.LevelDebug))
	require.NoError(t, err)
	require.NoError(t, NewHandler(ok).Handle(t.Context(), rec))
}

// flakyWriter fails until it is healed, like a destination that recovers.
type flakyWriter struct {
	failing atomic.Bool
	written atomic.Int64
}

func (w *flakyWriter) Write(p []byte) (int, error) {
	if w.failing.Load() {
		return 0, errWriteFailed
	}

	w.written.Add(int64(len(p)))

	return len(p), nil
}

// TestErrWriter_ErrorIsClearedOnceTaken pins that the remembered write error is taken, not merely
// read: a destination that fails once and then recovers must stop reporting the stale failure, or
// every later Handle would keep returning an error for a record that was written.
func TestErrWriter_ErrorIsClearedOnceTaken(t *testing.T) {
	t.Parallel()

	dest := &flakyWriter{}
	dest.failing.Store(true)

	cfg, err := logutil.NewConfig(logutil.WithOutWriter(dest), logutil.WithLevel(logutil.LevelDebug))
	require.NoError(t, err)

	h := NewHandler(cfg)
	rec := slog.NewRecord(time.Now(), logutil.LevelInfo, "m", 0)

	require.ErrorIs(t, h.Handle(t.Context(), rec), errWriteFailed, "a failed write must be reported")

	dest.failing.Store(false)

	require.NoError(t, h.Handle(t.Context(), rec), "a healed destination must not report the stale error")
	require.Positive(t, dest.written.Load(), "the record must have been written")
	require.NoError(t, h.Handle(t.Context(), rec), "and must keep reporting nothing")
}

// TestErrWriter_KeepsTheMostRecentError pins that a pending error is replaced by a later one rather
// than held: when a transient failure is followed by a fatal one, the fatal one is the useful
// diagnosis. Exact per-call attribution is not on offer under concurrency (see errWriter), so the
// freshest error is the most informative thing the field can hold.
func TestErrWriter_KeepsTheMostRecentError(t *testing.T) {
	t.Parallel()

	stale := errors.New("transient failure")

	ew := &errWriter{w: io.Discard}

	_, _ = ew.Write([]byte("a"))
	require.NoError(t, ew.takeErr(), "a successful write records nothing")

	ew.w = &failingWriter{}
	ew.err.Store(&stale)

	_, _ = ew.Write([]byte("b")) // fails while the transient one is still pending

	require.ErrorIs(t, ew.takeErr(), errWriteFailed, "the most recent failure must be the one reported")
	require.NoError(t, ew.takeErr(), "and must be cleared once taken")
}

// TestErrWriter_SequentialFailuresEachReported pins the guarantee that does hold without concurrency:
// every failed write is reported to the next Handle that looks, and a success reports nothing.
func TestErrWriter_SequentialFailuresEachReported(t *testing.T) {
	t.Parallel()

	ew := &errWriter{w: &failingWriter{}}

	for range 3 {
		_, _ = ew.Write([]byte("x"))
		require.ErrorIs(t, ew.takeErr(), errWriteFailed, "each failed write must be reported")
	}

	ew.w = io.Discard

	_, _ = ew.Write([]byte("x"))
	require.NoError(t, ew.takeErr(), "a healed destination reports nothing")
}

// TestErrWriter_NilReceiver pins that a hand-built handler with no destination reports no error.
func TestErrWriter_NilReceiver(t *testing.T) {
	t.Parallel()

	var ew *errWriter

	require.NoError(t, ew.takeErr())
}

// TestNewHandler_FieldOrder pins the field order the handler documents: on the ungrouped path the
// trace ID follows the record's attributes, and on the grouped path it precedes the open group.
func TestNewHandler_FieldOrder(t *testing.T) {
	t.Parallel()

	out := &bytes.Buffer{}

	cfg, err := logutil.NewConfig(
		logutil.WithOutWriter(out),
		logutil.WithFormat(logutil.FormatJSON),
		logutil.WithLevel(logutil.LevelDebug),
		logutil.WithTraceIDFn(func() string { return "INJECTED" }),
	)
	require.NoError(t, err)

	l := slog.New(NewHandler(cfg))

	l.Info("m", "rec", "R")

	s := out.String()
	require.Less(t, strings.Index(s, `"rec"`), strings.Index(s, `"trace_id"`),
		"ungrouped: the trace ID follows the record's attributes")

	out.Reset()
	l.WithGroup("g").Info("m", "rec", "R")

	s = out.String()
	require.Less(t, strings.Index(s, `"trace_id"`), strings.Index(s, `"g"`),
		"grouped: the trace ID precedes the open group")
}

// Test_writerByFormat_console covers the console writer separately: it carries a FormatTimestamp
// function, so the struct cannot be compared by equality.
func Test_writerByFormat_console(t *testing.T) {
	t.Parallel()

	// A non-terminal writer makes the expected NoColor value deterministic
	// (writing console output to a buffer/file must not embed ANSI escapes).
	consoleOut := &bytes.Buffer{}

	cw, ok := writerByFormat(logutil.FormatConsole, consoleOut).(zerolog.ConsoleWriter)
	require.True(t, ok, "FormatConsole must yield a zerolog.ConsoleWriter")
	require.Equal(t, consoleOut, cw.Out)
	require.True(t, cw.NoColor, "a non-terminal writer must not be colorized")
	require.NotNil(t, cw.FormatTimestamp,
		"the timestamp formatter must be set, so the console does not parse with zerolog's global TimeFieldFormat")
}

// Test_consoleTimestamp pins the console timestamp formatter: it parses the RFC3339Nano value the
// handler writes (whatever the process-global zerolog.TimeFieldFormat says), renders it in the local
// zone in zerolog's console time format, and mirrors zerolog's own formatter for degenerate inputs.
//
// The expected rendering is spelled out as a literal (rather than computed with the production
// constant, which no change to it could ever fail) by building the instant in the local zone.
func Test_consoleTimestamp(t *testing.T) {
	t.Parallel()

	local := time.Date(2026, 7, 12, 17, 58, 36, 490566713, time.Local) //nolint:gosmopolitan // pins the local-zone rendering.

	plain := consoleTimestamp(true)
	require.Equal(t, "5:58PM", plain(local.Format(timeLayoutBare)), "renders in zerolog's console time format (time.Kitchen)")

	// The same instant carried in another zone must render identically: the formatter converts to
	// the local zone, so a UTC-stamped record is not shown at a misleading clock time.
	require.Equal(t, "5:58PM", plain(local.UTC().Format(timeLayoutBare)), "the instant is converted to the local zone")

	require.Equal(t, "not a time", plain("not a time"), "an unparseable value is passed through verbatim")
	require.Equal(t, "<nil>", plain(nil), "a missing time field renders as zerolog's own placeholder")
	require.Equal(t, "1783000000", plain(json.Number("1783000000")),
		"a non-string value (a user attribute colliding with the reserved key) is passed through, not dropped")

	colored := consoleTimestamp(false)
	require.Equal(t, "\x1b[90m5:58PM\x1b[0m", colored(local.Format(timeLayoutBare)), "colorized like zerolog's own formatter")
}

// Test_consoleTimestamp_noColorEnv pins that the timestamp honors the NO_COLOR convention, as every
// other part of a zerolog console line does: without it the line would carry raw ANSI escapes on the
// timestamp alone. It cannot run in parallel, as it sets an environment variable.
func Test_consoleTimestamp_noColorEnv(t *testing.T) {
	t.Setenv(noColorEnv, "1")

	local := time.Date(2026, 7, 12, 17, 58, 36, 0, time.Local) //nolint:gosmopolitan // pins the local-zone rendering.

	require.Equal(t, "5:58PM", consoleTimestamp(false)(local.Format(timeLayoutBare)),
		"NO_COLOR must suppress the escape sequence even for a colorized writer")
}
