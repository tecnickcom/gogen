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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tecnickcom/nurago/pkg/logutil"
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

// TestNewLogger_hookReceivesOriginalLevel verifies the hook runs at the slog layer and is
// invoked with the original record level (Notice, Critical), which a zerolog-level hook could
// not see since the handler hands zerolog a NoLevel event.
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
// actively logging. The native handler holds no process-global state (the level
// mapping is a pure function), so construction and logging must be race-free.
// This test must pass under -race.
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

// TestNewHandler_concurrentWriterSafe verifies the handler serializes writes so a
// non-thread-safe cfg.Out (here a bytes.Buffer) is safe under concurrent logging, matching
// logutil's standard-library backend. It must pass under -race.
func TestNewHandler_concurrentWriterSafe(t *testing.T) {
	t.Parallel()

	cfg, err := logutil.NewConfig(
		logutil.WithOutWriter(&bytes.Buffer{}),
		logutil.WithFormat(logutil.FormatJSON),
		logutil.WithLevel(logutil.LevelInfo),
	)
	require.NoError(t, err)

	l := slog.New(NewHandler(cfg))

	var wg sync.WaitGroup

	wg.Add(16)

	for range 16 {
		go func() {
			defer wg.Done()

			for range 100 {
				l.Info("concurrent", "k", "v")
			}
		}()
	}

	wg.Wait()
}

// TestNewLogger_singleTimestamp guards against a record carrying a duplicate
// "time" field: the handler must stamp the record time exactly once.
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

// TestNewLogger_traceIDStaysAtRootUnderGroup verifies logutil's trace-ID handler keeps
// the trace ID at the root of the record even when the logger is derived with WithGroup.
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

// traceGroupValuer resolves to a group carrying a trace ID: under an empty key it inlines onto the
// root, so the trace ID only becomes visible once the value is resolved.
type traceGroupValuer struct{}

func (traceGroupValuer) LogValue() slog.Value {
	return slog.GroupValue(slog.String(logutil.TraceIDKey, "CALLER"))
}

// TestNewHandler_TraceIDNotDuplicated asserts that a caller-supplied trace_id landing at the root
// suppresses the injected one on every path that gets it there: a record attribute, With/WithAttrs,
// cfg.CommonAttr, an inlined (empty-key) group on either path (including one produced by a
// LogValuer), and a With followed by a WithGroup. A duplicate key would not be merely cosmetic: the
// injected value is written last, so a last-wins parser (Go's own encoding/json, jq, Elasticsearch)
// resolves trace_id to it and the caller's value is destroyed.
func TestNewHandler_TraceIDNotDuplicated(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		common []logutil.Attr
		log    func(l *slog.Logger)
		want   any // the value a last-wins decode must yield for the root trace_id
		keys   int // emitted trace_id keys (0 means 1: a second one may nest inside a group)
	}{
		{
			name: "record attribute",
			log:  func(l *slog.Logger) { l.Info("m", "trace_id", "CALLER") },
			want: "CALLER",
		},
		{
			name: "With",
			log:  func(l *slog.Logger) { l.With("trace_id", "CALLER").Info("m") },
			want: "CALLER",
		},
		{
			name:   "CommonAttr",
			common: []logutil.Attr{slog.String("trace_id", "CALLER")},
			log:    func(l *slog.Logger) { l.Info("m") },
			want:   "CALLER",
		},
		{
			name: "inlined group",
			log:  func(l *slog.Logger) { l.With(slog.Group("", slog.String("trace_id", "CALLER"))).Info("m") },
			want: "CALLER",
		},
		{
			// An inlined group is flattened onto the root, so a trace_id inside one is a root
			// trace_id even when it arrives as a record attribute.
			name: "record inlined group",
			log:  func(l *slog.Logger) { l.Info("m", slog.Group("", slog.String("trace_id", "CALLER"))) },
			want: "CALLER",
		},
		{
			// Same, but the group only exists once the LogValuer is resolved.
			name: "record LogValuer resolving to an inlined group",
			log:  func(l *slog.Logger) { l.Info("m", slog.Any("", traceGroupValuer{})) },
			want: "CALLER",
		},
		{
			name: "With then WithGroup",
			log:  func(l *slog.Logger) { l.With("trace_id", "CALLER").WithGroup("g").Info("m", "k", "v") },
			want: "CALLER",
		},
		{
			// A group opened under the reserved key writes a root trace_id field of its own (an
			// object), so it must suppress the injected one rather than collide with it.
			name: "WithGroup(trace_id)",
			log:  func(l *slog.Logger) { l.WithGroup("trace_id").Info("m", "a", "b") },
			want: map[string]any{"a": "b"},
		},
		{
			// Only the ROOT group takes the key's place: a trace_id group nested inside another
			// nests with it, so the root trace ID must still be injected — and the nested one is a
			// second key, at a different level, not a duplicate.
			name: "WithGroup(g) then WithGroup(trace_id)",
			log:  func(l *slog.Logger) { l.WithGroup("g").WithGroup("trace_id").Info("m", "a", "b") },
			want: "INJECTED",
			keys: 2,
		},
		{
			// Conversely, the root group named trace_id suppresses the injection even when further
			// groups nest inside it.
			name: "WithGroup(trace_id) then WithGroup(x)",
			log:  func(l *slog.Logger) { l.WithGroup("trace_id").WithGroup("x").Info("m", "a", "b") },
			want: map[string]any{"x": map[string]any{"a": "b"}},
		},
		{
			name: "no caller trace_id",
			log:  func(l *slog.Logger) { l.Info("m") },
			want: "INJECTED",
		},
		// The cases below pin the other half of the rule: only a field that is actually WRITTEN
		// counts. An elided one must not suppress the injection, or the record would end up with no
		// trace ID at all — worse than the duplicate the suppression exists to prevent.
		{
			name: "elided: empty group named trace_id",
			log:  func(l *slog.Logger) { l.With(slog.Group("trace_id")).Info("m") },
			want: "INJECTED",
		},
		{
			name: "elided: typed-nil error under trace_id",
			log:  func(l *slog.Logger) { l.Info("m", slog.Any("trace_id", (*typedNilError)(nil))) },
			want: "INJECTED",
		},
		{
			name: "elided: baked typed-nil error under trace_id",
			log:  func(l *slog.Logger) { l.With(slog.Any("trace_id", (*typedNilError)(nil))).Info("m") },
			want: "INJECTED",
		},
		{
			name: "elided: empty WithGroup(trace_id)",
			log:  func(l *slog.Logger) { l.WithGroup("trace_id").Info("m") },
			want: "INJECTED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			out := &bytes.Buffer{}

			cfg, err := logutil.NewConfig(
				logutil.WithOutWriter(out),
				logutil.WithFormat(logutil.FormatJSON),
				logutil.WithLevel(logutil.LevelDebug),
				logutil.WithCommonAttr(tt.common...),
				logutil.WithTraceIDFn(func() string { return "INJECTED" }),
			)
			require.NoError(t, err)

			tt.log(slog.New(NewHandler(cfg)))

			// One key at the root. A case may legitimately carry a second one nested inside a group,
			// where it is a distinct field at a distinct level rather than a duplicate.
			keys := tt.keys
			if keys == 0 {
				keys = 1
			}

			s := out.String()
			require.Equal(t, keys, strings.Count(s, `"trace_id":`), "unexpected trace_id key count in: %s", s)

			// Decoding with encoding/json is the last-wins check: with a duplicate key it would
			// yield the injected value regardless of what the caller supplied.
			m := map[string]any{}
			require.NoError(t, json.Unmarshal(out.Bytes(), &m), "output must be valid JSON: %s", s)
			require.Equal(t, tt.want, m["trace_id"])
		})
	}
}

// TestNewHandler_TraceIDFnNotCalledWhenSuppressed verifies that a trace ID baked in via
// With/WithAttrs short-circuits TraceIDFn entirely, on both the ungrouped and the grouped path.
func TestNewHandler_TraceIDFnNotCalledWhenSuppressed(t *testing.T) {
	t.Parallel()

	var calls atomic.Int64

	cfg, err := logutil.NewConfig(
		logutil.WithOutWriter(io.Discard),
		logutil.WithFormat(logutil.FormatJSON),
		logutil.WithLevel(logutil.LevelDebug),
		logutil.WithTraceIDFn(func() string {
			calls.Add(1)
			return "INJECTED"
		}),
	)
	require.NoError(t, err)

	l := slog.New(NewHandler(cfg)).With("trace_id", "CALLER")

	l.Info("ungrouped")
	l.WithGroup("g").Info("grouped", "k", "v")
	require.Equal(t, int64(0), calls.Load(), "TraceIDFn must not run when the trace ID is suppressed")

	slog.New(NewHandler(cfg)).Info("no caller trace_id")
	require.Equal(t, int64(1), calls.Load(), "TraceIDFn must run when it provides the trace ID")
}

// TestNewHandler_TraceIDUnderGroupNests pins the counterpart of the suppression rule: a trace_id
// supplied under an open group — as a record attribute or via WithAttrs/With — nests inside that
// group (standard slog semantics) and therefore does not suppress the root one. They are distinct
// fields at distinct levels, so neither is a duplicate key, and suppressing the root injection here
// would silently delete the record's only root-level trace ID.
func TestNewHandler_TraceIDUnderGroupNests(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		log  func(l *slog.Logger)
	}{
		{
			name: "record attribute",
			log:  func(l *slog.Logger) { l.WithGroup("g").Info("m", "trace_id", "NESTED") },
		},
		{
			name: "WithAttrs under the group",
			log:  func(l *slog.Logger) { l.WithGroup("g").With("trace_id", "NESTED").Info("m") },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			out := &bytes.Buffer{}

			cfg, err := logutil.NewConfig(
				logutil.WithOutWriter(out),
				logutil.WithFormat(logutil.FormatJSON),
				logutil.WithLevel(logutil.LevelDebug),
				logutil.WithTraceIDFn(func() string { return "INJECTED" }),
			)
			require.NoError(t, err)

			tt.log(slog.New(NewHandler(cfg)))

			m := map[string]any{}
			require.NoError(t, json.Unmarshal(out.Bytes(), &m), "output: %s", out)
			require.Equal(t, "INJECTED", m["trace_id"], "the root trace_id is still injected")

			g, ok := m["g"].(map[string]any)
			require.True(t, ok, "the group must be present")
			require.Equal(t, "NESTED", g["trace_id"], "a trace_id under an open group nests inside it")
		})
	}
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
