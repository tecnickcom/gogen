package logutil

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSlogTraceIDHandler_EmptyWithReturnsReceiver(t *testing.T) {
	t.Parallel()

	base := newSlogTraceIDHandler(slog.DiscardHandler, func() string { return "id" }, false)

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

// traceGroupValuer resolves to a group carrying a trace ID: under an empty key it inlines onto the
// root, so the trace ID only becomes visible once the value is resolved.
type traceGroupValuer struct{}

func (traceGroupValuer) LogValue() slog.Value {
	return slog.GroupValue(slog.String(TraceIDKey, "CALLER"))
}

// TestSlogTraceIDHandler_HandlerTraceIDNotDuplicated verifies that a trace ID supplied through the
// handler rather than the record — via CommonAttr, With/WithAttrs, or an inlined (empty-key) group —
// also suppresses the injected one. A duplicate key would put the injected value last, so a
// last-wins parser (encoding/json, jq, Elasticsearch) would resolve trace_id to it and silently
// destroy the caller's value.
func TestSlogTraceIDHandler_HandlerTraceIDNotDuplicated(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		common []Attr
		log    func(l *slog.Logger)
		want   any // the value a last-wins decode must yield for the root trace ID
		keys   int // emitted trace ID keys (0 means 1: a second one may nest inside a group)
	}{
		{
			name: "With",
			log:  func(l *slog.Logger) { l.With(TraceIDKey, "CALLER").Info("m") },
			want: "CALLER",
		},
		{
			name:   "CommonAttr",
			common: []Attr{slog.String(TraceIDKey, "CALLER")},
			log:    func(l *slog.Logger) { l.Info("m") },
			want:   "CALLER",
		},
		{
			name: "inlined group",
			log:  func(l *slog.Logger) { l.With(slog.Group("", slog.String(TraceIDKey, "CALLER"))).Info("m") },
			want: "CALLER",
		},
		{
			// An inlined group is flattened onto the root, so a trace ID inside one is a root-level
			// trace ID even when it arrives as a record attribute.
			name: "record inlined group",
			log:  func(l *slog.Logger) { l.Info("m", slog.Group("", slog.String(TraceIDKey, "CALLER"))) },
			want: "CALLER",
		},
		{
			name: "With then WithGroup",
			log:  func(l *slog.Logger) { l.With(TraceIDKey, "CALLER").WithGroup("g").Info("m", "k", "v") },
			want: "CALLER",
		},
		{
			// The trace ID only exists once the LogValuer is resolved. The scan resolves the few
			// attributes that could carry the key, so it is still seen: were it missed, the injected
			// key would be written as well — and last, destroying the caller's value.
			name: "LogValuer resolving to an inlined group",
			log:  func(l *slog.Logger) { l.Info("m", slog.Any("", traceGroupValuer{})) },
			want: "CALLER",
		},
		{
			name:   "CommonAttr LogValuer resolving to an inlined group",
			common: []Attr{slog.Any("", traceGroupValuer{})},
			log:    func(l *slog.Logger) { l.Info("m") },
			want:   "CALLER",
		},
		{
			name: "With LogValuer resolving to an inlined group",
			log:  func(l *slog.Logger) { l.With(slog.Any("", traceGroupValuer{})).Info("m") },
			want: "CALLER",
		},
		{
			// A group opened under the reserved key writes a root trace ID field of its own.
			name: "WithGroup(trace_id)",
			log:  func(l *slog.Logger) { l.WithGroup(TraceIDKey).Info("m", "a", "b") },
			want: map[string]any{"a": "b"},
		},
		{
			// Only the ROOT group takes the key's place: one nested inside another nests with it, so
			// the root trace ID is still injected (and the nested key is not a duplicate).
			name: "WithGroup(g) then WithGroup(trace_id)",
			log:  func(l *slog.Logger) { l.WithGroup("g").WithGroup(TraceIDKey).Info("m", "a", "b") },
			want: "INJECTED",
			keys: 2,
		},
		{
			name: "WithGroup(trace_id) then WithGroup(x)",
			log:  func(l *slog.Logger) { l.WithGroup(TraceIDKey).WithGroup("x").Info("m", "a", "b") },
			want: map[string]any{"x": map[string]any{"a": "b"}},
		},
		{
			// A root group named trace_id that renders nothing is elided by slog, so it cannot stand
			// in for the trace ID: the injected value must be written instead. Deciding this at
			// derivation time (where it is unknowable) would leave such records with no trace ID.
			name: "elided: empty WithGroup(trace_id)",
			log:  func(l *slog.Logger) { l.WithGroup(TraceIDKey).Info("m") },
			want: "INJECTED",
		},
		{
			name: "elided: WithGroup(trace_id) with only a zero attr",
			log:  func(l *slog.Logger) { l.WithGroup(TraceIDKey).Info("m", slog.Attr{}) }, //nolint:loggercheck // the zero Attr is a valid slog argument; the linter cannot model it.
			want: "INJECTED",
		},
		{
			// Attributes added under the group guarantee it renders, whatever the record carries.
			name: "WithGroup(trace_id) filled by With",
			log:  func(l *slog.Logger) { l.WithGroup(TraceIDKey).With("a", "b").Info("m") },
			want: map[string]any{"a": "b"},
		},
		{
			name: "elided: group named trace_id holding only a zero attr",
			log:  func(l *slog.Logger) { l.Info("m", slog.Group(TraceIDKey, slog.Attr{})) }, //nolint:loggercheck // the zero Attr is a valid slog argument; the linter cannot model it.
			want: "INJECTED",
		},
		{
			name: "elided: nested empty group named trace_id",
			log:  func(l *slog.Logger) { l.With(slog.Group(TraceIDKey, slog.Group("inner", slog.Attr{}))).Info("m") }, //nolint:loggercheck // the zero Attr is a valid slog argument; the linter cannot model it.
			want: "INJECTED",
		},
		{
			name: "no caller trace ID",
			log:  func(l *slog.Logger) { l.Info("m") },
			want: "INJECTED",
		},
		{
			// An empty group writes no field, so it must not suppress the injection: slog ignores it,
			// and the record would otherwise carry no trace ID at all.
			name: "elided: empty group named trace_id",
			log:  func(l *slog.Logger) { l.With(slog.Group(TraceIDKey)).Info("m") },
			want: "INJECTED",
		},
		{
			name:   "elided: empty CommonAttr group named trace_id",
			common: []Attr{slog.Group(TraceIDKey)},
			log:    func(l *slog.Logger) { l.Info("m") },
			want:   "INJECTED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer

			cfg, err := NewConfig(
				WithOutWriter(&buf),
				WithFormat(FormatJSON),
				WithCommonAttr(tt.common...),
				WithTraceIDFn(func() string { return "INJECTED" }),
			)
			require.NoError(t, err)

			tt.log(cfg.SlogLogger())

			// One key at the root. A case may legitimately carry a second one nested inside a group,
			// where it is a distinct field at a distinct level rather than a duplicate.
			keys := tt.keys
			if keys == 0 {
				keys = 1
			}

			out := buf.String()
			require.Equal(t, keys, strings.Count(out, `"`+TraceIDKey+`":`), "unexpected trace ID key count in: %s", out)

			// Decoding is the last-wins check: with a duplicate key it would yield the injected
			// value whatever the caller supplied.
			var got map[string]any
			require.NoError(t, json.Unmarshal(buf.Bytes(), &got), "output must be valid JSON: %s", out)
			require.Equal(t, tt.want, got[TraceIDKey])
		})
	}
}

// TestSlogTraceIDHandler_TraceIDUnderGroupNests pins the counterpart of the suppression rule: a
// trace ID supplied under an open group — as a record attribute or via With/WithAttrs — nests inside
// that group and must not suppress the root injection, or the record would be left with no
// root-level trace ID at all.
func TestSlogTraceIDHandler_TraceIDUnderGroupNests(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		log  func(l *slog.Logger)
	}{
		{
			name: "record attribute",
			log:  func(l *slog.Logger) { l.WithGroup("g").Info("m", TraceIDKey, "NESTED") },
		},
		{
			name: "WithAttrs under the group",
			log:  func(l *slog.Logger) { l.WithGroup("g").With(TraceIDKey, "NESTED").Info("m") },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer

			cfg, err := NewConfig(
				WithOutWriter(&buf),
				WithFormat(FormatJSON),
				WithTraceIDFn(func() string { return "INJECTED" }),
			)
			require.NoError(t, err)

			tt.log(cfg.SlogLogger())

			var got map[string]any
			require.NoError(t, json.Unmarshal(buf.Bytes(), &got), "output: %s", buf.String())
			require.Equal(t, "INJECTED", got[TraceIDKey], "the root trace ID is still injected")

			group, ok := got["g"].(map[string]any)
			require.True(t, ok, "the group must be present")
			require.Equal(t, "NESTED", group[TraceIDKey], "a trace ID under an open group nests inside it")
		})
	}
}

// Test_valueRenders covers the rule behind both the trace ID suppression and the elided-group
// removal: a value renders unless it is a group that yields no field, however deeply the emptiness
// is nested. A value that renders nothing writes no field, so it can neither stand in for the
// injected trace ID nor be left in the record (see sanitizeRecord).
func Test_valueRenders(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		attr Attr
		want bool
	}{
		{name: "scalar", attr: slog.String("k", "v"), want: true},
		{name: "group with a field", attr: slog.Group("g", slog.Int("n", 1)), want: true},
		{name: "group with no attributes", attr: slog.Group("g"), want: false},
		{name: "group holding only the zero attr", attr: Attr{Key: "g", Value: slog.GroupValue(Attr{})}, want: false},
		{
			name: "group holding only an empty subgroup",
			attr: Attr{Key: "g", Value: slog.GroupValue(Attr{Key: "in", Value: slog.GroupValue(Attr{})})},
			want: false,
		},
		{
			name: "group holding an empty subgroup and a field",
			attr: Attr{Key: "g", Value: slog.GroupValue(Attr{Key: "in", Value: slog.GroupValue()}, slog.Int("n", 1))},
			want: true,
		},
		{name: "LogValuer resolving to a scalar", attr: slog.Any("k", countingValuer{new(atomic.Int64)}), want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.want, valueRenders(tt.attr.Value.Resolve()))

			// attrsRender and recordRenders answer the same question for a list and a record, and
			// both skip the zero Attr, which slog drops.
			attrs := []Attr{{}, tt.attr}
			require.Equal(t, tt.want, attrsRender(attrs))

			rec := slog.NewRecord(time.Time{}, LevelInfo, "m", 0)
			rec.AddAttrs(attrs...)
			require.Equal(t, tt.want, recordRenders(rec))
		})
	}

	require.False(t, attrsRender(nil), "no attributes render nothing")
	require.False(t, recordRenders(slog.NewRecord(time.Time{}, LevelInfo, "m", 0)), "an attribute-less record renders nothing")
}

// Test_hasRootKey covers the root-level key scan behind the trace ID deduplication: an inlined
// (empty-key) group is flattened onto the enclosing level and so is descended into, while a named
// group nests its members under its own key and so is not. Values are deliberately not resolved.
func Test_hasRootKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		attrs []Attr
		want  bool
	}{
		{
			name:  "root attribute after a non-matching one",
			attrs: []Attr{slog.Int("n", 1), slog.String(TraceIDKey, "x")},
			want:  true,
		},
		{
			name:  "named group does not land at the root",
			attrs: []Attr{slog.Group("g", slog.String(TraceIDKey, "x"))},
			want:  false,
		},
		{
			name:  "inlined group is flattened onto the root",
			attrs: []Attr{slog.Int("n", 1), slog.Group("", slog.String(TraceIDKey, "x"))},
			want:  true,
		},
		{
			name:  "nested inlined groups",
			attrs: []Attr{slog.Group("", slog.Group("", slog.String(TraceIDKey, "x")))},
			want:  true,
		},
		{
			name:  "inlined group without the key",
			attrs: []Attr{slog.Group("", slog.String("other", "x"))},
			want:  false,
		},
		{
			name:  "LogValuer resolving to an inlined group with the key",
			attrs: []Attr{slog.Any("", traceGroupValuer{})},
			want:  true,
		},
		{
			// slog ignores a group with no attributes even under a non-empty key: it writes no field,
			// so it must not be reported as carrying the key.
			name:  "empty group named with the key",
			attrs: []Attr{slog.Group(TraceIDKey)},
			want:  false,
		},
		{
			name:  "empty-key non-group value",
			attrs: []Attr{{Key: "", Value: slog.StringValue("v")}},
			want:  false,
		},
		{
			name:  "zero attr",
			attrs: []Attr{{}},
			want:  false,
		},
		{
			name:  "no attributes",
			attrs: nil,
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.want, hasRootKey(tt.attrs, TraceIDKey))
		})
	}
}

// countingValuer counts how many times its value is resolved.
type countingValuer struct{ n *atomic.Int64 }

func (c countingValuer) LogValue() slog.Value {
	c.n.Add(1)

	return slog.StringValue("resolved")
}

// TestSlogTraceIDHandler_ResolvesEachValueOnce pins the cost model of the whole chain: a value is
// resolved exactly once per record, whatever key it is logged under — including the reserved key and
// an empty (inlining) one, which the trace-ID scan has to look inside.
//
// The sanitizing handler resolves each value and hands the *resolved* attribute downstream, so every
// later pass — the trace-ID scan, and the standard library handler as it writes — re-resolves an
// already-resolved value, which is a no-op. A LogValuer's side effects therefore fire once, and a
// valuer whose result varies between calls cannot make one pass disagree with another.
func TestSlogTraceIDHandler_ResolvesEachValueOnce(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		key  string
	}{
		{name: "an ordinary key"},
		{name: "the reserved key", key: TraceIDKey},
		{name: "an empty key, which might inline"},
	}
	tests[0].key = "k"

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var (
				buf bytes.Buffer
				n   atomic.Int64
			)

			cfg, err := NewConfig(
				WithOutWriter(&buf),
				WithFormat(FormatJSON),
				WithTraceIDFn(func() string { return "INJECTED" }),
			)
			require.NoError(t, err)

			cfg.SlogLogger().Info("m", slog.Any(tt.key, countingValuer{&n}))

			require.Equal(t, int64(1), n.Load(), "the value must be resolved exactly once: %s", buf.String())
		})
	}
}

// fanoutHandler hands the same record to several handlers, as tee handlers (slog-multi) do. Copies
// of a slog.Record share state, so a handler that mutates one without Record.Clone corrupts the
// others — the standard library detects it and writes a "!BUG" attribute into the line.
type fanoutHandler struct{ handlers []slog.Handler }

func (f fanoutHandler) Enabled(context.Context, slog.Level) bool { return true }
func (f fanoutHandler) Handle(ctx context.Context, record slog.Record) error {
	for _, h := range f.handlers {
		_ = h.Handle(ctx, record) // a test tee: every sink gets the record.
	}

	return nil
}
func (f fanoutHandler) WithAttrs([]slog.Attr) slog.Handler { return f }
func (f fanoutHandler) WithGroup(string) slog.Handler      { return f }

// addAttrHandler adds an attribute to the record before passing it on, as request-context middleware
// (slog-context, otelslog) does. It is what leaves the record with the spare capacity that makes a
// later unsafe mutation observable.
type addAttrHandler struct{ inner slog.Handler }

func (a addAttrHandler) Enabled(context.Context, slog.Level) bool { return true }
func (a addAttrHandler) Handle(ctx context.Context, record slog.Record) error {
	record.AddAttrs(slog.String("req_id", "R1"))

	return a.inner.Handle(ctx, record) //nolint:wrapcheck
}
func (a addAttrHandler) WithAttrs([]slog.Attr) slog.Handler { return a }
func (a addAttrHandler) WithGroup(string) slog.Handler      { return a }

// TestSlogTraceIDHandler_DoesNotMutateSharedRecord pins that the handler leaves the caller's record
// alone: it injects the trace ID into a rebuilt record instead. Mutating a record whose copies have
// been handed out breaks slog's contract, and the standard library announces it by writing a "!BUG"
// field into the log line — through a middleware that adds attributes and a tee to two sinks, the
// composition this test builds.
func TestSlogTraceIDHandler_DoesNotMutateSharedRecord(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	cfg, err := NewConfig(
		WithOutWriter(&buf),
		WithFormat(FormatJSON),
		WithTraceIDFn(func() string { return "TID" }),
	)
	require.NoError(t, err)

	l := slog.New(addAttrHandler{fanoutHandler{[]slog.Handler{cfg.SlogHandler(), cfg.SlogHandler()}}})
	l.Info("request done", "a", 1, "b", 2, "c", 3, "d", 4, "e", 5, "f", 6, "g", 7)

	out := buf.String()
	require.NotContains(t, out, "!BUG", "the record must not be mutated in place: %s", out)
	require.Equal(t, 2, strings.Count(out, `"`+TraceIDKey+`":"TID"`), "both sinks carry the trace ID")

	for line := range strings.Lines(strings.TrimSpace(out)) {
		var got map[string]any
		require.NoError(t, json.Unmarshal([]byte(line), &got), "every line must be valid JSON: %s", line)
		require.Equal(t, "R1", got["req_id"], "the middleware's attribute must survive")
	}
}

// TestSlogTraceIDHandler_ElidedGroupKeepsOutputValid pins that a record carrying a group which
// renders nothing still produces valid JSON. The standard library's JSON handler rolls its buffer
// back for such a group without restoring the separator, so the next attribute is emitted with no
// comma before it; the trace ID this handler adds would be exactly that attribute. Groups that
// render nothing are therefore dropped from the record, which changes no output.
func TestSlogTraceIDHandler_ElidedGroupKeepsOutputValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		log  func(l *slog.Logger)
	}{
		{
			name: "elided group alone",
			log:  func(l *slog.Logger) { l.Info("m", slog.Group("g", slog.Attr{})) }, //nolint:loggercheck // the zero Attr is a valid slog argument; the linter cannot model it.
		},
		{
			name: "elided group followed by an attribute",
			log:  func(l *slog.Logger) { l.Info("m", slog.Group("g", slog.Attr{}), slog.String("k", "v")) }, //nolint:loggercheck // the zero Attr is a valid slog argument; the linter cannot model it.
		},
		{
			name: "nested elided group",
			log:  func(l *slog.Logger) { l.Info("m", slog.Group("g", slog.Group("inner", slog.Attr{}))) }, //nolint:loggercheck // the zero Attr is a valid slog argument; the linter cannot model it.
		},
		{
			name: "elided group under an open group",
			log: func(l *slog.Logger) {
				l.WithGroup("open").Info("m", slog.Group("g", slog.Attr{}), slog.String("k", "v")) //nolint:loggercheck // the zero Attr is a valid slog argument; the linter cannot model it.
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer

			cfg, err := NewConfig(
				WithOutWriter(&buf),
				WithFormat(FormatJSON),
				WithTraceIDFn(func() string { return "TID" }),
			)
			require.NoError(t, err)

			tt.log(cfg.SlogLogger())

			var got map[string]any
			require.NoError(t, json.Unmarshal(buf.Bytes(), &got), "output must be valid JSON: %s", buf.String())
			require.Equal(t, "TID", got[TraceIDKey], "the trace ID must still be written")
			require.NotContains(t, got, "g", "a group that renders nothing must not appear")
		})
	}
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

	// A non-nil TraceIDFunc returns a *slogTraceIDHandler behind the sanitizing handler, which
	// filters the records and derivations it injects the trace ID into.
	sanitizer, wrapped := NewSlogTraceIDHandler(base, func() string { return "id" }).(*slogSanitizeHandler)
	require.True(t, wrapped, "non-nil TraceIDFunc must wrap the handler in the sanitizing handler")

	_, wrapped = sanitizer.inner.(*slogTraceIDHandler)
	require.True(t, wrapped, "non-nil TraceIDFunc must wrap the handler in the trace handler")
}

// TestNewSlogTraceIDHandler_NilHandler verifies the nil-handler guard: the constructor falls back to
// the default logger's handler rather than returning one that panics on first use.
func TestNewSlogTraceIDHandler_NilHandler(t *testing.T) { //nolint:paralleltest // mutates the slog default.
	var buf bytes.Buffer

	prev := slog.Default()

	defer slog.SetDefault(prev)

	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))

	slog.New(NewSlogTraceIDHandler(nil, func() string { return "TID" })).Info("m")

	require.Contains(t, buf.String(), `"trace_id":"TID"`, "a nil handler must fall back to the default one")
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

	base := newSlogTraceIDHandler(slog.DiscardHandler, func() string { return "id" }, false)

	got := base.WithAttrs([]slog.Attr{slog.String("a", "b")})

	traced, ok := got.(*slogTraceIDHandler)
	require.True(t, ok, "WithAttrs must return a *slogTraceIDHandler that preserves the trace ID func")
	require.NotNil(t, traced.traceIDFn)
}

func TestSlogTraceIDHandler_WithGroup(t *testing.T) {
	t.Parallel()

	base := newSlogTraceIDHandler(slog.DiscardHandler, func() string { return "id" }, false)

	got := base.WithGroup("g")

	traced, ok := got.(*slogTraceIDHandler)
	require.True(t, ok, "WithGroup must return a *slogTraceIDHandler that preserves the trace ID func")
	require.NotNil(t, traced.traceIDFn)
}
