package logsrv

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tecnickcom/gogen/pkg/logutil"
)

func TestNewLogger(t *testing.T) {
	t.Parallel()

	attr := []logutil.Attr{
		slog.String("program", "test"),
		slog.Int("version", 1),
	}

	var hookValue string

	hookFn := func(_ logutil.LogLevel, message string) {
		hookValue = message
	}

	cfg, err := logutil.NewConfig(
		logutil.WithOutWriter(os.Stderr),
		logutil.WithFormat(logutil.FormatJSON),
		logutil.WithLevel(logutil.LevelDebug),
		logutil.WithCommonAttr(attr...),
		logutil.WithHookFn(hookFn),
	)

	require.NoError(t, err)
	require.NotNil(t, cfg)

	l := NewLogger(cfg)

	require.NotNil(t, l)

	l.Info("test")

	require.Equal(t, "test", hookValue)
}

// TestNewLogger_nilTraceIDFn verifies that a nil TraceIDFn is treated as
// valid (as logutil does): the logger is created without panicking and the
// trace ID field is simply omitted from the output.
func TestNewLogger_nilTraceIDFn(t *testing.T) {
	t.Parallel()

	out := &bytes.Buffer{}

	cfg, err := logutil.NewConfig(
		logutil.WithOutWriter(out),
		logutil.WithFormat(logutil.FormatJSON),
		logutil.WithLevel(logutil.LevelDebug),
		logutil.WithTraceIDFn(nil),
	)

	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.Nil(t, cfg.TraceIDFn)

	require.NotPanics(t, func() {
		l := NewLogger(cfg)

		require.NotNil(t, l)

		l.Info("no trace id")
	})

	require.Contains(t, out.String(), "no trace id")
	require.NotContains(t, out.String(), logutil.TraceIDKey, "the trace ID field must be omitted when TraceIDFn is nil")
}

// TestNewLogger_hookReceivesOriginalLevel verifies the hook is invoked with
// the original slog record level instead of the zerolog-collapsed one
// (zerolog maps Notice->Info and Critical->Error).
func TestNewLogger_hookReceivesOriginalLevel(t *testing.T) {
	t.Parallel()

	type hookCall struct {
		level   logutil.LogLevel
		message string
	}

	var calls []hookCall

	hookFn := func(level logutil.LogLevel, message string) {
		calls = append(calls, hookCall{level: level, message: message})
	}

	cfg, err := logutil.NewConfig(
		logutil.WithOutWriter(io.Discard),
		logutil.WithFormat(logutil.FormatJSON),
		logutil.WithLevel(logutil.LevelDebug),
		logutil.WithHookFn(hookFn),
	)

	require.NoError(t, err)
	require.NotNil(t, cfg)

	l := NewLogger(cfg)

	require.NotNil(t, l)

	l.Log(t.Context(), logutil.LevelNotice, "notice message")
	l.Log(t.Context(), logutil.LevelCritical, "critical message")

	require.Equal(t, []hookCall{
		{level: logutil.LevelNotice, message: "notice message"},
		{level: logutil.LevelCritical, message: "critical message"},
	}, calls, "the hook must receive the original slog levels, not the zerolog-collapsed ones")
}

// TestNewLogger_concurrent creates many loggers concurrently while each one is
// actively logging. Before the sync.Once fix, NewLogger reassigned the global
// szlog.LogLevels map on every call while previously created handlers read it
// at log time, which the race detector flags. This test must pass under -race.
func TestNewLogger_concurrent(t *testing.T) {
	t.Parallel()

	const goroutines = 16

	cfg, err := logutil.NewConfig(
		logutil.WithOutWriter(io.Discard),
		logutil.WithFormat(logutil.FormatJSON),
		logutil.WithLevel(logutil.LevelDebug),
	)

	require.NoError(t, err)
	require.NotNil(t, cfg)

	var wg sync.WaitGroup

	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()

			l := NewLogger(cfg)

			assert.NotNil(t, l)

			for range 50 {
				l.Info("concurrent log line")
				l.Debug("concurrent debug line")
			}
		}()
	}

	wg.Wait()
}

// TestNewLogger_singleTimestamp guards against the regression where the zerolog
// context and the slog-zerolog handler each stamped a "time" field, producing a
// record with a duplicate JSON key.
func TestNewLogger_singleTimestamp(t *testing.T) {
	t.Parallel()

	out := &bytes.Buffer{}

	cfg, err := logutil.NewConfig(
		logutil.WithOutWriter(out),
		logutil.WithFormat(logutil.FormatJSON),
		logutil.WithLevel(logutil.LevelDebug),
	)
	require.NoError(t, err)

	l := NewLogger(cfg)
	l.Info("one ts")

	require.Equal(t, 1, strings.Count(out.String(), `"time":`), "each record must carry exactly one time field")
}

// TestNewLogger_traceIDPerRecord verifies the trace ID is resolved per record
// (matching logutil) rather than frozen at construction time.
func TestNewLogger_traceIDPerRecord(t *testing.T) {
	t.Parallel()

	out := &bytes.Buffer{}

	var n atomic.Int64

	cfg, err := logutil.NewConfig(
		logutil.WithOutWriter(out),
		logutil.WithFormat(logutil.FormatJSON),
		logutil.WithLevel(logutil.LevelDebug),
		logutil.WithTraceIDFn(func() string {
			return "trace-" + strconv.FormatInt(n.Add(1), 10)
		}),
	)
	require.NoError(t, err)

	l := NewLogger(cfg)
	l.Info("first")
	l.Info("second")

	require.Contains(t, out.String(), `"trace_id":"trace-1"`)
	require.Contains(t, out.String(), `"trace_id":"trace-2"`, "the trace ID must be re-resolved for every record")
}

// TestNewLogger_traceIDStaysAtRootUnderGroup verifies the converter keeps the trace ID
// at the root of the record even when the logger is derived with WithGroup.
func TestNewLogger_traceIDStaysAtRootUnderGroup(t *testing.T) {
	t.Parallel()

	out := &bytes.Buffer{}

	cfg, err := logutil.NewConfig(
		logutil.WithOutWriter(out),
		logutil.WithFormat(logutil.FormatJSON),
		logutil.WithLevel(logutil.LevelDebug),
		logutil.WithTraceIDFn(func() string { return "trace-root" }),
	)
	require.NoError(t, err)

	l := NewLogger(cfg).WithGroup("g")
	l.Info("msg", "k", "v")

	var got map[string]any
	require.NoError(t, json.Unmarshal(out.Bytes(), &got))

	require.Equal(t, "trace-root", got[logutil.TraceIDKey], "trace_id must stay at the root, not nest in the group")

	group, ok := got["g"].(map[string]any)
	require.True(t, ok, "the group must be present")
	require.NotContains(t, group, logutil.TraceIDKey, "trace_id must not be nested inside the group")
}

func TestNewHandler_FormatNoneNoHookIsDiscard(t *testing.T) {
	t.Parallel()

	cfg, err := logutil.NewConfig(logutil.WithFormat(logutil.FormatNone))
	require.NoError(t, err)

	h := NewHandler(cfg)
	require.False(t, h.Enabled(t.Context(), logutil.LevelError), "FormatNone without a hook must be a zero-cost discard handler")
}

func TestNewLogger_FormatNoneHookFires(t *testing.T) {
	t.Parallel()

	var fired int

	cfg, err := logutil.NewConfig(
		logutil.WithFormat(logutil.FormatNone),
		logutil.WithLevel(logutil.LevelDebug),
		logutil.WithHookFn(func(_ logutil.LogLevel, _ string) { fired++ }),
	)
	require.NoError(t, err)

	l := NewLogger(cfg)
	l.Error("x")
	l.Info("y")

	require.Equal(t, 2, fired, "hooks must fire under FormatNone even though output is discarded")
}

func TestNewLogger_UserTraceIDKept(t *testing.T) {
	t.Parallel()

	out := &bytes.Buffer{}

	cfg, err := logutil.NewConfig(
		logutil.WithOutWriter(out),
		logutil.WithFormat(logutil.FormatJSON),
		logutil.WithLevel(logutil.LevelDebug),
		logutil.WithTraceIDFn(func() string { return "CONFIGURED" }),
	)
	require.NoError(t, err)

	NewLogger(cfg).Info("m", "trace_id", "USER")

	s := out.String()
	require.Equal(t, 1, strings.Count(s, `"trace_id":`), "exactly one trace_id key")
	require.Contains(t, s, `"trace_id":"USER"`, "the caller-supplied trace_id wins")
}

func TestNewLogger_Source(t *testing.T) {
	t.Parallel()

	out := &bytes.Buffer{}

	cfg, err := logutil.NewConfig(
		logutil.WithOutWriter(out),
		logutil.WithFormat(logutil.FormatJSON),
		logutil.WithLevel(logutil.LevelDebug),
		logutil.WithSource(true),
	)
	require.NoError(t, err)

	NewLogger(cfg).Info("with source")

	require.Contains(t, out.String(), `"source":`, "source location must be present when enabled")
}

func Test_isTerminalWriter(t *testing.T) {
	t.Parallel()

	require.False(t, isTerminalWriter(&bytes.Buffer{}), "a non-file writer is not a terminal")

	f, err := os.CreateTemp(t.TempDir(), "logsrv")
	require.NoError(t, err)

	defer func() { _ = f.Close() }()

	require.False(t, isTerminalWriter(f), "a regular file is not a terminal")
}

// TestNewLogger_severeLevelsDoNotTerminate locks in the invariant that the
// Emergency->panic and Alert->fatal mappings emit ordinary records (via zerolog's
// WithLevel) and never terminate the process.
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

	require.Contains(t, out.String(), `"level":"panic"`)
	require.Contains(t, out.String(), `"level":"fatal"`)
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

// TestNewHandler_nilOut verifies a nil Out writer falls back to os.Stderr instead
// of building a handler that panics on first write.
func TestNewHandler_nilOut(t *testing.T) {
	t.Parallel()

	cfg, err := logutil.NewConfig(logutil.WithFormat(logutil.FormatJSON))
	require.NoError(t, err)

	cfg.Out = nil // hand-cleared writer must fall back, not panic

	require.NotPanics(t, func() {
		h := NewHandler(cfg)
		require.NotNil(t, h)
	})
}

func Test_writerByFormat(t *testing.T) {
	t.Parallel()

	// A non-terminal writer makes the expected NoColor value deterministic
	// (writing console output to a buffer/file must not embed ANSI escapes).
	consoleOut := &bytes.Buffer{}

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
			name:   "console",
			format: logutil.FormatConsole,
			out:    consoleOut,
			want:   zerolog.ConsoleWriter{Out: consoleOut, NoColor: true},
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
