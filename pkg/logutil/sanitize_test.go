package logutil

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// emptyGroupValuer resolves to a group with no members at all, which writes no field.
type emptyGroupValuer struct{}

func (emptyGroupValuer) LogValue() slog.Value { return slog.GroupValue() }

// sparseValuer is the shape that defeats a purely structural elision filter: an "omitempty" LogValuer
// whose absent fields become the zero Attr, so an empty one resolves to a group with members that all
// elide. Nothing about the *unresolved* attribute says so — its kind is KindLogValuer, not KindGroup —
// yet the standard library resolves it, finds a group that renders nothing, and rolls the buffer back
// without restoring the separator. It is an ordinary idiom, not a contrived one.
type sparseValuer struct{ id, name string }

func (u sparseValuer) LogValue() slog.Value {
	attrs := make([]slog.Attr, 0, 2)

	if u.id != "" {
		attrs = append(attrs, slog.String("id", u.id))
	} else {
		attrs = append(attrs, slog.Attr{})
	}

	if u.name != "" {
		attrs = append(attrs, slog.String("name", u.name))
	} else {
		attrs = append(attrs, slog.Attr{})
	}

	return slog.GroupValue(attrs...)
}

// sanitizeLogger builds a JSON logger writing into buf, with the given options applied.
func sanitizeLogger(t *testing.T, buf *bytes.Buffer, opts ...Option) *slog.Logger {
	t.Helper()

	cfg, err := NewConfig(append([]Option{WithOutWriter(buf), WithFormat(FormatJSON)}, opts...)...)
	require.NoError(t, err)

	return cfg.SlogLogger()
}

// traceIDFn returns a fixed trace ID.
func traceIDFn() string { return "TID" }

// TestSanitize_ElidingLogValuerKeepsOutputValid pins the class a structural (non-resolving) filter
// cannot see: a LogValuer that resolves to a group whose members all elide. It is the same defect as
// the literal elided group — a line with a missing comma — but reached through a value whose kind
// gives no hint of it, so the filter has to resolve to find it.
//
// Every route an attribute can take is covered, because each one lands in a different buffer: a record
// attribute, one nested in a group, one in the middle of a group that does render, a derived logger's
// baked attributes (which would poison every record it writes), and Config.CommonAttr (which would
// poison every line the process writes). With the trace ID turned off too, since that removes the trace
// handler from the chain entirely.
func TestSanitize_ElidingLogValuerKeepsOutputValid(t *testing.T) {
	t.Parallel()

	elider := slog.Any("v", sparseValuer{})

	tests := []struct {
		name string
		opts []Option
		log  func(l *slog.Logger)
		want map[string]any
	}{
		{
			name: "a record attribute",
			opts: []Option{WithTraceIDFn(traceIDFn)},
			log:  func(l *slog.Logger) { l.Info("m", elider, slog.Int("status", 200)) },
			want: map[string]any{"status": 200.0, TraceIDKey: "TID"},
		},
		{
			name: "nested in a group",
			opts: []Option{WithTraceIDFn(traceIDFn)},
			log:  func(l *slog.Logger) { l.Info("m", slog.Group("g", elider), slog.String("z", "1")) },
			want: map[string]any{"z": "1", TraceIDKey: "TID"},
		},
		{
			name: "in the middle of a group that renders",
			opts: []Option{WithTraceIDFn(traceIDFn)},
			log: func(l *slog.Logger) {
				l.Info("m", slog.Group("a", slog.String("k", "v"), slog.Group("b", elider), slog.String("k2", "v2")))
			},
			want: map[string]any{"a": map[string]any{"k": "v", "k2": "v2"}, TraceIDKey: "TID"},
		},
		{
			name: "baked into a derived logger",
			opts: []Option{WithTraceIDFn(traceIDFn)},
			log:  func(l *slog.Logger) { l.With(slog.String("a", "1"), elider, slog.String("z", "1")).Info("m") },
			want: map[string]any{"a": "1", "z": "1", TraceIDKey: "TID"},
		},
		{
			name: "in the common attributes",
			opts: []Option{WithTraceIDFn(traceIDFn), WithCommonAttr(slog.String("service", "svc"), elider, slog.String("env", "prod"))},
			log:  func(l *slog.Logger) { l.Info("m") },
			want: map[string]any{"service": "svc", "env": "prod", TraceIDKey: "TID"},
		},
		{
			name: "with the trace ID turned off",
			opts: []Option{WithTraceIDFn(nil)},
			log:  func(l *slog.Logger) { l.Info("m", elider, slog.String("z", "1")) },
			want: map[string]any{"z": "1"},
		},
		{
			name: "a valuer that still renders is untouched",
			opts: []Option{WithTraceIDFn(traceIDFn)},
			log: func(l *slog.Logger) {
				l.Info("m", slog.Any("user", sparseValuer{id: "u-1"}), slog.String("z", "1"))
			},
			want: map[string]any{"user": map[string]any{"id": "u-1"}, "z": "1", TraceIDKey: "TID"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer

			tt.log(sanitizeLogger(t, &buf, tt.opts...))

			line := bytes.TrimSpace(buf.Bytes())
			require.True(t, json.Valid(line), "the line must be valid JSON: %s", line)

			var got map[string]any

			require.NoError(t, json.Unmarshal(line, &got))

			for k, v := range tt.want {
				require.Equal(t, v, got[k], "field %q", k)
			}

			require.NotContains(t, got, "v", "a valuer that renders nothing must not appear")
		})
	}
}

// TestSanitize_CommonAttrPoisoningIsPerProcess pins that the common attributes are filtered once, at
// construction: they are preformatted into the handler in a single WithAttrs call, so one eliding
// group among them would leave EVERY line the process writes missing a comma — not just the first.
func TestSanitize_CommonAttrPoisoningIsPerProcess(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	l := sanitizeLogger(t, &buf, WithTraceIDFn(traceIDFn), WithCommonAttr(
		slog.String("service", "svc"), slog.Any("v", sparseValuer{}), slog.String("env", "prod")))

	for i := range 3 {
		buf.Reset()
		l.Info("record")

		require.True(t, json.Valid(bytes.TrimSpace(buf.Bytes())), "record %d must be valid JSON: %s", i, buf.String())
	}
}

// TestSanitize_ConsoleFormatKeepsKeysIntact pins the same defect in the text handler, where it does not
// break the syntax but silently RENAMES the next field: the elided group's name is left in the key
// prefix, so "z=1" is written as "g.v.z=1". No JSON-validity gate can catch that, which is exactly why
// it needs its own assertion — and why the filter has to run for FormatConsole too.
func TestSanitize_ConsoleFormatKeepsKeysIntact(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		attrs []any
	}{
		{name: "a literal eliding group", attrs: []any{slog.Group("g", slog.Attr{}), slog.String("z", "1")}}, //nolint:loggercheck // the zero Attr is a valid slog argument; the linter cannot model it.
		{name: "an eliding LogValuer", attrs: []any{slog.Group("g", slog.Any("v", sparseValuer{})), slog.String("z", "1")}},
		{name: "an eliding LogValuer at the root", attrs: []any{slog.Any("v", sparseValuer{}), slog.String("z", "1")}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer

			cfg, err := NewConfig(WithOutWriter(&buf), WithFormat(FormatConsole), WithTraceIDFn(traceIDFn))
			require.NoError(t, err)

			cfg.SlogLogger().Info("m", tt.attrs...)

			line := buf.String()
			require.Contains(t, line, " z=1", "the attribute after the elided group must keep its own key: %s", line)
			require.NotContains(t, line, "g.", "no key may inherit the elided group's prefix: %s", line)
			require.Contains(t, line, "trace_id=TID", "the trace ID must still be written: %s", line)
		})
	}
}

// TestSanitize_EmptySourceElided pins that an empty *slog.Source writes no field, as slog's own
// handlers have it. It is not a curiosity: such a value under the reserved trace ID key would otherwise
// be taken for a caller-supplied trace ID and suppress the injected one while writing nothing, leaving
// the record with no trace ID at all.
func TestSanitize_EmptySourceElided(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		log  func(l *slog.Logger)
	}{
		{
			name: "as a record attribute",
			log:  func(l *slog.Logger) { l.Info("m", slog.Any(TraceIDKey, (*slog.Source)(nil))) },
		},
		{
			name: "baked into a derived logger",
			log:  func(l *slog.Logger) { l.With(slog.Any(TraceIDKey, (*slog.Source)(nil))).Info("m") },
		},
		{
			name: "a zero-valued Source",
			log:  func(l *slog.Logger) { l.Info("m", slog.Any(TraceIDKey, &slog.Source{})) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer

			tt.log(sanitizeLogger(t, &buf, WithTraceIDFn(traceIDFn)))

			var got map[string]any

			require.NoError(t, json.Unmarshal(buf.Bytes(), &got), "the line must be valid JSON: %s", buf.String())
			require.Equal(t, "TID", got[TraceIDKey],
				"an empty Source writes nothing, so it cannot stand in for the trace ID: %s", buf.String())
		})
	}
}

// TestSanitize_NonEmptySourceStillRenders guards the other side of the Source rule: a real caller
// location is a field like any other and must not be elided.
func TestSanitize_NonEmptySourceStillRenders(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	sanitizeLogger(t, &buf, WithTraceIDFn(traceIDFn)).
		Info("m", slog.Any("caller", &slog.Source{Function: "f", File: "f.go", Line: 12}))

	var got map[string]any

	require.NoError(t, json.Unmarshal(buf.Bytes(), &got), "the line must be valid JSON: %s", buf.String())
	require.Contains(t, got, "caller", "a non-empty Source must be written")
}

// TestSanitize_ElidedGroupKeepsOutputValid pins that a group rendering no field never leaves the line
// unparseable, wherever it appears. The standard library's handlers roll the output buffer back past
// such a group without restoring the separator they cleared when they opened it, so the next attribute
// is written with no comma before it — and the trace ID this package injects is very often that next
// attribute.
//
// The filter therefore has to reach every place an attribute can be written, not just a record's top
// level: through Config.CommonAttr (preformatted in a single WithAttrs call, so one elided group there
// makes *every* line the process writes unparseable), through With/WithAttrs, nested inside a group
// that is itself kept, under an open group, and with the trace ID turned off entirely — which leaves
// no trace handler in the chain at all.
func TestSanitize_ElidedGroupKeepsOutputValid(t *testing.T) {
	t.Parallel()

	elided := slog.Group("e", slog.Attr{}) //nolint:loggercheck // the zero Attr is a valid slog argument; the linter cannot model it.

	tests := []struct {
		name string
		opts []Option
		log  func(l *slog.Logger)
		want map[string]any // the fields that must survive, beyond the built-ins
	}{
		{
			name: "common attributes",
			opts: []Option{WithTraceIDFn(traceIDFn), WithCommonAttr(slog.String("service", "svc"), elided, slog.String("env", "prod"))},
			log:  func(l *slog.Logger) { l.Info("m") },
			want: map[string]any{"service": "svc", "env": "prod", TraceIDKey: "TID"},
		},
		{
			name: "baked with With",
			opts: []Option{WithTraceIDFn(traceIDFn)},
			log: func(l *slog.Logger) {
				l.With(slog.String("a", "1"), elided, slog.String("b", "2")).Info("m")
			},
			want: map[string]any{"a": "1", "b": "2", TraceIDKey: "TID"},
		},
		{
			name: "nested inside a group that renders",
			opts: []Option{WithTraceIDFn(traceIDFn)},
			log: func(l *slog.Logger) {
				l.Info("m", slog.Group("g", slog.String("a", "1"), elided, slog.String("b", "2")))
			},
			want: map[string]any{"g": map[string]any{"a": "1", "b": "2"}, TraceIDKey: "TID"},
		},
		{
			name: "record attribute with the trace ID turned off",
			opts: []Option{WithTraceIDFn(nil)},
			log:  func(l *slog.Logger) { l.Info("m", elided, slog.String("z", "1")) },
			want: map[string]any{"z": "1"},
		},
		{
			name: "a group that needs no filtering, beside one that does",
			opts: []Option{WithTraceIDFn(traceIDFn)},
			log: func(l *slog.Logger) {
				l.Info("m", slog.Group("keep", slog.String("a", "1")), elided, slog.String("z", "1"))
			},
			want: map[string]any{"keep": map[string]any{"a": "1"}, "z": "1", TraceIDKey: "TID"},
		},
		{
			name: "caller-supplied trace ID, which suppresses the injected one",
			opts: []Option{WithTraceIDFn(traceIDFn)},
			log: func(l *slog.Logger) {
				l.Info("m", slog.String(TraceIDKey, "CALLER"), elided, slog.String("z", "1"))
			},
			want: map[string]any{"z": "1", TraceIDKey: "CALLER"},
		},
		{
			name: "under an open group",
			opts: []Option{WithTraceIDFn(traceIDFn)},
			log: func(l *slog.Logger) {
				l.WithGroup("open").Info("m", elided, slog.String("z", "1"))
			},
			want: map[string]any{"open": map[string]any{"z": "1"}, TraceIDKey: "TID"},
		},
		{
			name: "baked under an open group",
			opts: []Option{WithTraceIDFn(traceIDFn)},
			log: func(l *slog.Logger) {
				l.WithGroup("open").With(slog.String("a", "1"), elided, slog.String("b", "2")).Info("m")
			},
			want: map[string]any{"open": map[string]any{"a": "1", "b": "2"}, TraceIDKey: "TID"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer

			tt.log(sanitizeLogger(t, &buf, tt.opts...))

			line := bytes.TrimSpace(buf.Bytes())
			require.True(t, json.Valid(line), "the line must be valid JSON: %s", line)

			var got map[string]any

			require.NoError(t, json.Unmarshal(line, &got))

			for k, v := range tt.want {
				require.Equal(t, v, got[k], "field %q", k)
			}

			require.NotContains(t, got, "e", "a group that renders nothing must not appear")
		})
	}
}

// TestSanitize_LogValuerResolvedOnce pins that deciding whether a group renders does not resolve its
// members: the handler downstream resolves them again as it writes them, so resolving here would fire
// every LogValuer's side effects twice per record — metrics counters, lazy fetches, one-shot
// decrypt/redact, audit hooks — and a valuer whose result varied between the two resolutions would
// make the elision decision and the write disagree.
func TestSanitize_LogValuerResolvedOnce(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		log  func(l *slog.Logger, v countingValuer)
	}{
		{
			name: "inside a group",
			log:  func(l *slog.Logger, v countingValuer) { l.Info("m", slog.Group("g", slog.Any("v", v))) },
		},
		{
			name: "at the record root",
			log:  func(l *slog.Logger, v countingValuer) { l.Info("m", slog.Any("v", v)) },
		},
		{
			name: "nested two groups deep",
			log: func(l *slog.Logger, v countingValuer) {
				l.Info("m", slog.Group("g", slog.Group("h", slog.Any("v", v))))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var (
				buf bytes.Buffer
				n   atomic.Int64
			)

			tt.log(sanitizeLogger(t, &buf, WithTraceIDFn(traceIDFn)), countingValuer{n: &n})

			require.Equal(t, int64(1), n.Load(),
				"the value must be resolved exactly once per record: %s", buf.String())
			require.Contains(t, buf.String(), `"v":"resolved"`)
		})
	}
}

// TestSanitize_ValueResolvingToEmptyGroup pins the one deliberate divergence from the standard
// library. A member that *resolves* to an empty group writes no field, but the standard library still
// counts it as rendered, so it emits the enclosing group as a bare "{}". Both of this package's
// backends elide it instead, which is the rule they document everywhere else ("a group that renders
// nothing is elided") and the only rule under which they agree with each other.
//
// The stakes are not cosmetic: it is the group's rendering that suppresses the injected trace ID, so
// counting a bare "{}" as a trace ID would ship the record with "trace_id":{} and destroy the
// correlation ID the logger exists to carry.
func TestSanitize_ValueResolvingToEmptyGroup(t *testing.T) {
	t.Parallel()

	t.Run("the enclosing group is elided, not written as a bare object", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer

		sanitizeLogger(t, &buf, WithTraceIDFn(traceIDFn)).
			Info("m", slog.Group("g", slog.Any("v", emptyGroupValuer{})), slog.Int("a", 1))

		var got map[string]any

		require.NoError(t, json.Unmarshal(buf.Bytes(), &got), "the line must be valid JSON: %s", buf.String())
		require.NotContains(t, got, "g", "a group whose only member writes nothing must be elided")
		require.InDelta(t, 1.0, got["a"], 0, "the attribute after it must survive")
	})

	t.Run("a root group named trace_id must not destroy the trace ID", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer

		sanitizeLogger(t, &buf, WithTraceIDFn(traceIDFn)).
			WithGroup(TraceIDKey).Info("m", slog.Any("v", emptyGroupValuer{}))

		line := buf.String()
		require.Equal(t, 1, countKeys(line, TraceIDKey), "exactly one root trace_id must be written: %s", line)

		var got map[string]any

		require.NoError(t, json.Unmarshal(buf.Bytes(), &got), "the line must be valid JSON: %s", line)
		require.Equal(t, "TID", got[TraceIDKey],
			"the group renders nothing, so the injected trace ID must stand rather than be replaced by {}")
	})
}

// countKeys counts the occurrences of a JSON key in a line.
func countKeys(line, key string) int {
	return strings.Count(line, `"`+key+`":`)
}

// TestSanitize_TraceGroupStillFilledByRenderingAttrs pins the traceFilled rule: attributes baked under
// a root group named trace_id guarantee that group renders — so it supplies the root trace ID, and the
// injected one must be suppressed — but only when they actually render something. A baked group that
// renders nothing leaves the decision to the record, and a record that renders nothing must still
// carry a trace ID rather than none at all.
func TestSanitize_TraceGroupStillFilledByRenderingAttrs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		derive  func(l *slog.Logger) *slog.Logger
		wantTID any // the value of the single root trace_id
	}{
		{
			name:    "a rendering baked attribute fills the group",
			derive:  func(l *slog.Logger) *slog.Logger { return l.With(slog.String("a", "1")) },
			wantTID: map[string]any{"a": "1"},
		},
		{
			name:    "a baked group that renders nothing does not",
			derive:  func(l *slog.Logger) *slog.Logger { return l.With(slog.Group("empty")) },
			wantTID: "TID",
		},
		{
			name: "nor does a baked zero attribute",
			//nolint:loggercheck // the zero Attr is a valid slog argument; the linter cannot model it.
			derive:  func(l *slog.Logger) *slog.Logger { return l.With(slog.Attr{}) },
			wantTID: "TID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer

			tt.derive(sanitizeLogger(t, &buf, WithTraceIDFn(traceIDFn)).WithGroup(TraceIDKey)).Info("m")

			line := buf.String()
			require.Equal(t, 1, countKeys(line, TraceIDKey), "exactly one root trace_id: %s", line)

			var got map[string]any

			require.NoError(t, json.Unmarshal(buf.Bytes(), &got), "the line must be valid JSON: %s", line)
			require.Equal(t, tt.wantTID, got[TraceIDKey])
		})
	}
}

// TestSanitize_HandlerContract pins the slog.Handler contract for the sanitizing wrapper.
func TestSanitize_HandlerContract(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	h := newSlogSanitizeHandler(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	require.Same(t, h, h.WithAttrs(nil), "empty WithAttrs must return the receiver")
	require.Same(t, h, h.WithGroup(""), "empty WithGroup must return the receiver")
	require.Same(t, h, h.WithAttrs([]Attr{slog.Group("empty")}),
		"WithAttrs left empty by the filter bakes nothing, so it must return the receiver too")
	require.NotSame(t, h, h.WithGroup("g"))
	require.True(t, h.Enabled(t.Context(), slog.LevelInfo))
	require.False(t, h.Enabled(t.Context(), slog.LevelDebug))
}

// TestSanitize_OutOfRangeTimeKeepsOutputValid pins the second shape the standard library encodes
// incorrectly: a time.Time whose year falls outside [0,9999]. slog's appendJSONTime writes an "!ERROR:"
// string for it and then writes the value anyway — it does not return after the error — so one key
// carries two JSON strings and the line is unparseable.
//
// It is not an exotic input: a deadline built by adding a large duration, or a time.Unix on a corrupt
// or attacker-controlled epoch field, lands here. Every route is covered, because each lands in a
// different buffer — and via Config.CommonAttr it would corrupt every line the process writes.
func TestSanitize_OutOfRangeTimeKeepsOutputValid(t *testing.T) {
	t.Parallel()

	far := time.Date(30000, 1, 2, 3, 4, 5, 6, time.UTC)
	wantFar := "30000-01-02T03:04:05.000000006Z"

	tests := []struct {
		name string
		opts []Option
		log  func(l *slog.Logger)
		key  string
		want string
	}{
		{
			name: "a record attribute",
			opts: []Option{WithTraceIDFn(traceIDFn)},
			log:  func(l *slog.Logger) { l.Info("m", slog.Time("exp", far), slog.Int("status", 200)) },
			key:  "exp",
			want: wantFar,
		},
		{
			name: "a negative year",
			opts: []Option{WithTraceIDFn(traceIDFn)},
			log: func(l *slog.Logger) {
				l.Info("m", slog.Time("exp", time.Date(-1, 1, 2, 3, 4, 5, 0, time.UTC)), slog.Int("status", 200))
			},
			key:  "exp",
			want: "-0001-01-02T03:04:05Z",
		},
		{
			name: "nested in a group",
			opts: []Option{WithTraceIDFn(traceIDFn)},
			log:  func(l *slog.Logger) { l.Info("m", slog.Group("g", slog.Time("exp", far))) },
			key:  "",
			want: "",
		},
		{
			name: "baked into a derived logger",
			opts: []Option{WithTraceIDFn(traceIDFn)},
			log:  func(l *slog.Logger) { l.With(slog.Time("exp", far), slog.String("z", "1")).Info("m") },
			key:  "exp",
			want: wantFar,
		},
		{
			name: "in the common attributes",
			opts: []Option{WithTraceIDFn(traceIDFn), WithCommonAttr(slog.Time("boot", far), slog.String("env", "prod"))},
			log:  func(l *slog.Logger) { l.Info("m") },
			key:  "boot",
			want: wantFar,
		},
		{
			name: "under the reserved trace ID key",
			opts: []Option{WithTraceIDFn(traceIDFn)},
			log:  func(l *slog.Logger) { l.Info("m", slog.Time(TraceIDKey, far)) },
			key:  TraceIDKey,
			want: wantFar,
		},
		{
			name: "delivered by a LogValuer",
			opts: []Option{WithTraceIDFn(traceIDFn)},
			log:  func(l *slog.Logger) { l.Info("m", slog.Any("exp", farTimeValuer{far})) },
			key:  "exp",
			want: wantFar,
		},
		{
			name: "an in-range time is left exactly as slog writes it",
			opts: []Option{WithTraceIDFn(traceIDFn)},
			log: func(l *slog.Logger) {
				l.Info("m", slog.Time("t", time.Date(2026, 7, 12, 1, 2, 3, 456789, time.UTC)))
			},
			key:  "t",
			want: "2026-07-12T01:02:03.000456789Z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer

			tt.log(sanitizeLogger(t, &buf, tt.opts...))

			line := bytes.TrimSpace(buf.Bytes())
			require.True(t, json.Valid(line), "the line must be valid JSON: %s", line)
			require.NotContains(t, string(line), "!ERROR:", "the value must be written as a string, not as an error")

			var got map[string]any

			require.NoError(t, json.Unmarshal(line, &got))

			if tt.key != "" {
				require.Equal(t, tt.want, got[tt.key], "field %q", tt.key)
			}
		})
	}
}

// farTimeValuer resolves to a time slog's JSON encoder cannot write, so the filter has to resolve to
// find it — the value's own kind is KindLogValuer, not KindTime.
type farTimeValuer struct{ t time.Time }

func (f farTimeValuer) LogValue() slog.Value { return slog.TimeValue(f.t) }

// nilPtrError is an error on a pointer receiver, so a nil one is a typed nil: non-nil as an interface,
// nil underneath. It is the commonest error bug in Go, and slog renders it as the string "<nil>" where
// zerolog — and therefore this package's sibling backend — writes no field at all.
type nilPtrError struct{}

func (e *nilPtrError) Error() string {
	if e == nil {
		return "no-error"
	}

	return "err"
}

// sliceError is an aggregate error whose underlying kind is a slice (the shape of
// validator.ValidationErrors). A nil one is NOT a nil pointer, so zerolog calls Error() on it and writes
// the field — it must keep rendering.
type sliceError []string

func (s sliceError) Error() string { return "validation: " + strconv.Itoa(len(s)) }

// TestSanitize_NilPointerErrorElided pins that a typed-nil error writes no field, matching zerolog's
// AnErr — and so the logsrv backend — rather than slog, which renders it as the string "<nil>".
//
// The field itself is the smaller half. Under the reserved trace ID key, a rendered "<nil>" would be
// read as a caller-supplied trace ID and suppress the injected one, shipping the record correlated by
// the string "<nil>"; and a group whose only member is such a value must be elided rather than left as a
// bare "{}", exactly as the sibling backend elides it.
func TestSanitize_NilPointerErrorElided(t *testing.T) {
	t.Parallel()

	var e *nilPtrError

	tests := []struct {
		name    string
		opts    []Option
		log     func(l *slog.Logger)
		absent  []string
		present map[string]any
	}{
		{
			name:    "a record attribute writes no field",
			opts:    []Option{WithTraceIDFn(traceIDFn)},
			log:     func(l *slog.Logger) { l.Info("m", slog.Any("err", e), slog.Int("a", 1)) },
			absent:  []string{"err"},
			present: map[string]any{"a": 1.0, TraceIDKey: "TID"},
		},
		{
			name:    "a group holding only one is elided",
			opts:    []Option{WithTraceIDFn(traceIDFn)},
			log:     func(l *slog.Logger) { l.Info("m", slog.Group("g", slog.Any("err", e)), slog.Int("a", 1)) },
			absent:  []string{"g", "err"},
			present: map[string]any{"a": 1.0, TraceIDKey: "TID"},
		},
		{
			name:    "under the reserved key it does not suppress the injected trace ID",
			opts:    []Option{WithTraceIDFn(traceIDFn)},
			log:     func(l *slog.Logger) { l.Info("m", slog.Any(TraceIDKey, e)) },
			present: map[string]any{TraceIDKey: "TID"},
		},
		{
			name:    "baked into a derived logger",
			opts:    []Option{WithTraceIDFn(traceIDFn)},
			log:     func(l *slog.Logger) { l.With(slog.Any("err", e), slog.String("z", "1")).Info("m") },
			absent:  []string{"err"},
			present: map[string]any{"z": "1", TraceIDKey: "TID"},
		},
		{
			name:    "baked under the reserved key",
			opts:    []Option{WithTraceIDFn(traceIDFn)},
			log:     func(l *slog.Logger) { l.With(slog.Any(TraceIDKey, e)).Info("m") },
			present: map[string]any{TraceIDKey: "TID"},
		},
		{
			name:    "in the common attributes",
			opts:    []Option{WithTraceIDFn(traceIDFn), WithCommonAttr(slog.Any("err", e), slog.String("svc", "s"))},
			log:     func(l *slog.Logger) { l.Info("m") },
			absent:  []string{"err"},
			present: map[string]any{"svc": "s", TraceIDKey: "TID"},
		},
		{
			name:    "an ordinary error still renders",
			opts:    []Option{WithTraceIDFn(traceIDFn)},
			log:     func(l *slog.Logger) { l.Info("m", slog.Any("err", errors.New("boom"))) },
			present: map[string]any{"err": "boom", TraceIDKey: "TID"},
		},
		{
			name:    "a nil aggregate error still renders",
			opts:    []Option{WithTraceIDFn(traceIDFn)},
			log:     func(l *slog.Logger) { l.Info("m", slog.Any("err", sliceError(nil))) },
			present: map[string]any{"err": "validation: 0", TraceIDKey: "TID"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer

			tt.log(sanitizeLogger(t, &buf, tt.opts...))

			line := bytes.TrimSpace(buf.Bytes())
			require.True(t, json.Valid(line), "the line must be valid JSON: %s", line)
			require.Equal(t, 1, strings.Count(string(line), `"`+TraceIDKey+`":`), "exactly one root trace_id: %s", line)

			var got map[string]any

			require.NoError(t, json.Unmarshal(line, &got))

			for _, k := range tt.absent {
				require.NotContains(t, got, k, "a nil-pointer error must write no field")
			}

			for k, v := range tt.present {
				require.Equal(t, v, got[k], "field %q", k)
			}
		})
	}
}

// TestSanitize_TimeYearBoundaries pins the exact range slog's JSON encoder accepts. The rewrite rule is
// a boundary test (year < 10000), and an off-by-one in either direction is invisible to every other
// test: too wide and year 10000 emits two JSON strings under one key, too narrow and an ordinary
// timestamp is needlessly turned into a string.
func TestSanitize_TimeYearBoundaries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		year    int
		rewrite bool // whether the value must be rewritten (i.e. slog cannot encode it)
	}{
		{name: "year -1 is rewritten", year: -1, rewrite: true},
		{name: "year 0 is encodable", year: 0, rewrite: false},
		{name: "year 1 is encodable", year: 1, rewrite: false},
		{name: "year 9999 is encodable", year: 9999, rewrite: false},
		{name: "year 10000 is rewritten", year: 10000, rewrite: true},
		{name: "year 10001 is rewritten", year: 10001, rewrite: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ts := time.Date(tt.year, 1, 2, 3, 4, 5, 0, time.UTC)

			require.Equal(t, !tt.rewrite, encodableJSONTime(ts))

			var buf bytes.Buffer

			sanitizeLogger(t, &buf, WithTraceIDFn(traceIDFn)).Info("m", slog.Time("t", ts), slog.Int("z", 1))

			line := bytes.TrimSpace(buf.Bytes())
			require.True(t, json.Valid(line), "the line must be valid JSON whatever the year: %s", line)
			require.NotContains(t, string(line), "!ERROR:", "slog must never be handed a time it cannot encode")

			var got map[string]any

			require.NoError(t, json.Unmarshal(line, &got))
			require.Equal(t, ts.Format(time.RFC3339Nano), got["t"], "the instant must survive either way")
		})
	}
}

// TestSanitize_OutOfRangeRecordTimestamp pins the record's OWN timestamp, which is not an attribute and
// so never passes through the filter: it is repaired by the ReplaceAttr callback instead (see
// replaceLevelName). slog.Logger always stamps time.Now(), so only a hand-built record — from a
// middleware, a tee, or a replay tool — can carry a year slog cannot encode.
func TestSanitize_OutOfRangeRecordTimestamp(t *testing.T) {
	t.Parallel()

	for _, year := range []int{30000, -1} {
		t.Run(strconv.Itoa(year), func(t *testing.T) {
			t.Parallel()

			ts := time.Date(year, 1, 2, 3, 4, 5, 0, time.UTC)

			var buf bytes.Buffer

			cfg, err := NewConfig(WithOutWriter(&buf), WithFormat(FormatJSON), WithTraceIDFn(traceIDFn))
			require.NoError(t, err)

			rec := slog.NewRecord(ts, LevelInfo, "m", 0)
			rec.AddAttrs(slog.Int("a", 1))

			require.NoError(t, cfg.SlogHandler().Handle(context.Background(), rec))

			line := bytes.TrimSpace(buf.Bytes())
			require.True(t, json.Valid(line), "an out-of-range record timestamp must not break the line: %s", line)
			require.NotContains(t, string(line), "!ERROR:")

			var got map[string]any

			require.NoError(t, json.Unmarshal(line, &got))
			require.Equal(t, ts.Format(time.RFC3339Nano), got[slog.TimeKey])
			require.InDelta(t, 1.0, got["a"], 0, "the record's attributes must survive")
		})
	}
}
