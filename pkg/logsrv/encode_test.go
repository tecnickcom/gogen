package logsrv

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"runtime"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"github.com/tecnickcom/nurago/pkg/logutil"
)

func Test_zerologHandler_errorRendersAsString(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	rec := makeRecord(logutil.LevelError, "m",
		slog.Any("error", errors.New("boom")),
		slog.Any("err", errors.New("bang")),
		slog.Any("whatever", errors.New("kaboom")),
	)
	require.NoError(t, newLeaf(buf).Handle(context.Background(), rec))

	m := decodeJSON(t, buf.Bytes())
	require.Equal(t, "boom", m["error"], "an error must render as its message string, not a structured object")
	require.Equal(t, "bang", m["err"])
	require.Equal(t, "kaboom", m["whatever"], "the rendering is chosen by the value's type: the key is irrelevant")
}

// typedNilError is an error implementation used to exercise the typed-nil error value.
type typedNilError struct{}

func (*typedNilError) Error() string { return "typed nil" }

// Test_zerologHandler_typedNilErrorOmitted pins the documented behavior of a typed-nil error value
// (a nil *typedNilError, non-nil as an interface): zerolog's AnErr reads it as no error and writes no
// field at all, so it is dropped rather than rendered as null (unlike an untyped nil, above).
func Test_zerologHandler_typedNilErrorOmitted(t *testing.T) {
	t.Parallel()

	var e *typedNilError

	// Record-attribute path.
	buf := &bytes.Buffer{}
	require.NoError(t, newLeaf(buf).Handle(context.Background(), makeRecord(logutil.LevelInfo, "m", slog.Any("err", e))))
	require.NotContains(t, decodeJSON(t, buf.Bytes()), "err", "a typed-nil error must be omitted, not rendered")

	// Baked (WithAttrs) path.
	buf.Reset()
	h := newLeaf(buf).WithAttrs([]slog.Attr{slog.Any("err", e)})
	require.NoError(t, h.Handle(context.Background(), makeRecord(logutil.LevelInfo, "m")))
	require.NotContains(t, decodeJSON(t, buf.Bytes()), "err", "a typed-nil error must be omitted on the baked path too")
}

// Test_zerologHandler_typedNilErrorElidesGroup pins that a dropped value takes its enclosing group
// with it: a group whose only member is a typed-nil error produces no field, so the group is elided
// rather than rendered as a bare "{}" — the elision rule counts fields actually written.
func Test_zerologHandler_typedNilErrorElidesGroup(t *testing.T) {
	t.Parallel()

	var e *typedNilError

	// Record-attribute path: the group is elided, the sibling attribute survives.
	buf := &bytes.Buffer{}
	rec := makeRecord(logutil.LevelInfo, "m", grp("g", slog.Any("err", e)), slog.String("keep", "yes"))
	require.NoError(t, newLeaf(buf).Handle(context.Background(), rec))

	m := decodeJSON(t, buf.Bytes())
	require.NotContains(t, m, "g", "a group left with no fields must be elided, not rendered as {}")
	require.Equal(t, "yes", m["keep"])

	// Baked (WithAttrs) path.
	buf.Reset()
	h := newLeaf(buf).WithAttrs([]slog.Attr{grp("g", slog.Any("err", e))})
	require.NoError(t, h.Handle(context.Background(), makeRecord(logutil.LevelInfo, "m")))
	require.NotContains(t, decodeJSON(t, buf.Bytes()), "g", "the baked path must elide it too")

	// Under an open group, the same rule elides the open group itself.
	buf.Reset()
	hg := newLeaf(buf).WithGroup("outer")
	require.NoError(t, hg.Handle(context.Background(), makeRecord(logutil.LevelInfo, "m", slog.Any("err", e))))
	require.NotContains(t, decodeJSON(t, buf.Bytes()), "outer", "an open group left with no fields must be elided")
}

// Test_zerologHandler_elisionPropagatesThroughGroups pins that elision propagates up more than one
// level: a group whose sole member is an *elided* (not merely empty) group produces no field itself,
// so it must be elided too rather than rendered as a bare "{}". slog.GroupValue already strips
// directly-empty groups, so the only way to build this is a group holding a dropped value.
func Test_zerologHandler_elisionPropagatesThroughGroups(t *testing.T) {
	t.Parallel()

	var e *typedNilError

	nested := grp("outer", grp("inner", slog.Any("err", e)))

	// Record-attribute path.
	buf := &bytes.Buffer{}
	require.NoError(t, newLeaf(buf).Handle(context.Background(), makeRecord(logutil.LevelInfo, "m", nested)))
	require.NotContains(t, decodeJSON(t, buf.Bytes()), "outer", "elision must propagate through the enclosing group")

	// Baked (WithAttrs) path.
	buf.Reset()
	require.NoError(t, newLeaf(buf).WithAttrs([]slog.Attr{nested}).
		Handle(context.Background(), makeRecord(logutil.LevelInfo, "m")))
	require.NotContains(t, decodeJSON(t, buf.Bytes()), "outer", "the baked path must propagate it too")

	// Under an open group.
	buf.Reset()
	require.NoError(t, newLeaf(buf).WithGroup("open").
		Handle(context.Background(), makeRecord(logutil.LevelInfo, "m", nested)))
	require.NotContains(t, decodeJSON(t, buf.Bytes()), "open", "the open group must be elided as well")
}

// countingValuer counts how many times its value is resolved.
type countingValuer struct{ n *atomic.Int64 }

func (c countingValuer) LogValue() slog.Value {
	c.n.Add(1)

	return slog.StringValue("resolved")
}

// Test_zerologHandler_resolvesValueOnce pins the single-pass invariant the handler is built around:
// detection of a root trace_id happens as the attribute is written, so no value is ever resolved a
// second time. A LogValuer with side effects (a metrics counter, a lazy fetch) must fire once.
func Test_zerologHandler_resolvesValueOnce(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		attr slog.Attr
	}{
		{name: "plain key", attr: slog.Any("k", nil)},
		{name: "empty key", attr: slog.Any("", nil)},
		{name: "the trace_id key", attr: slog.Any(logutil.TraceIDKey, nil)},
		{name: "inside a named group", attr: grp("g", slog.Any("k", nil))},
		{name: "inside an inlined group", attr: grp("", slog.Any("k", nil))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Record-attribute path.
			var n atomic.Int64

			attr := tt.attr
			setValuer(&attr, countingValuer{&n})

			h := &zerologHandler{
				logger:    zerolog.New(io.Discard).Level(zerolog.TraceLevel),
				minLevel:  logutil.LevelTrace,
				traceIDFn: func() string { return "INJECTED" },
			}
			require.NoError(t, h.Handle(context.Background(), makeRecord(logutil.LevelInfo, "m", attr)))
			require.Equal(t, int64(1), n.Load(), "a record attribute must be resolved exactly once")

			// Baked (WithAttrs) path: resolved once at derivation, never again per record.
			n.Store(0)

			baked := h.WithAttrs([]slog.Attr{attr})

			require.Equal(t, int64(1), n.Load(), "a baked attribute must be resolved exactly once, at WithAttrs time")

			require.NoError(t, baked.Handle(context.Background(), makeRecord(logutil.LevelInfo, "m")))
			require.NoError(t, baked.Handle(context.Background(), makeRecord(logutil.LevelInfo, "m")))
			require.Equal(t, int64(1), n.Load(), "a baked attribute must not be re-resolved per record")
		})
	}
}

// setValuer replaces the placeholder value in a (possibly grouped) attribute with the given LogValuer.
func setValuer(a *slog.Attr, v slog.LogValuer) {
	if a.Value.Kind() == slog.KindGroup {
		members := a.Value.Group()
		clone := make([]slog.Attr, len(members))
		copy(clone, members)
		setValuer(&clone[0], v)

		a.Value = slog.GroupValue(clone...)

		return
	}

	a.Value = slog.AnyValue(v)
}

// valueError is an error implemented on a value type (not a pointer), so it can never be a typed
// nil — it exercises isNilError's non-nilable branch.
type valueError struct{}

func (valueError) Error() string { return "value error" }

// sliceError is an aggregate error whose underlying kind is a slice — the shape of
// validator.ValidationErrors. A nil one is NOT a typed nil for zerolog: it renders its message.
type sliceError []string

func (e sliceError) Error() string { return strconv.Itoa(len(e)) + " validation errors" }

// mapError is an aggregate error whose underlying kind is a map (e.g. gorilla/schema.MultiError).
type mapError map[string]string

func (e mapError) Error() string { return strconv.Itoa(len(e)) + " field errors" }
func Test_isNilError(t *testing.T) {
	t.Parallel()

	var (
		typedNil  *typedNilError
		nilSlice  sliceError
		nilMap    mapError
		nilAsAny  error = typedNil
		emptyMapE       = mapError{}
	)

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil interface", err: nil, want: true},
		{name: "typed nil pointer", err: typedNil, want: true},
		{name: "typed nil pointer as interface", err: nilAsAny, want: true},
		{name: "non-nil pointer", err: &typedNilError{}, want: false},
		{name: "non-nilable value type", err: valueError{}, want: false},
		{name: "plain error", err: errors.New("boom"), want: false},
		// zerolog's isNilValue only treats a pointer as nilable: a nil slice- or map-kind error is
		// rendered by AnErr, so it must not be reported as nil here (or the field would vanish).
		{name: "nil slice-kind error", err: nilSlice, want: false},
		{name: "nil map-kind error", err: nilMap, want: false},
		{name: "empty map-kind error", err: emptyMapE, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.want, isNilError(tt.err))
		})
	}
}

// Test_zerologHandler_nilAggregateErrorRendered pins that a nil aggregate error (a nil slice- or
// map-kind value implementing error) is RENDERED as its message, not dropped: zerolog's AnErr calls
// Error() on it, so dropping it here would silently lose a field the stdlib backend writes — and,
// since the elision rule keys off the same decision, would take the enclosing group with it.
func Test_zerologHandler_nilAggregateErrorRendered(t *testing.T) {
	t.Parallel()

	var (
		nilSlice sliceError
		nilMap   mapError
	)

	// Record-attribute path, at the root and inside a group.
	buf := &bytes.Buffer{}
	rec := makeRecord(logutil.LevelInfo, "m",
		slog.Any("errs", nilSlice),
		grp("g", slog.Any("fields", nilMap)),
	)
	require.NoError(t, newLeaf(buf).Handle(context.Background(), rec))

	m := decodeJSON(t, buf.Bytes())
	require.Equal(t, "0 validation errors", m["errs"], "a nil slice-kind error renders as its message")

	g, ok := m["g"].(map[string]any)
	require.True(t, ok, "the group must survive: its member wrote a field")
	require.Equal(t, "0 field errors", g["fields"])

	// Baked (WithAttrs) path.
	buf.Reset()
	h := newLeaf(buf).WithAttrs([]slog.Attr{slog.Any("errs", nilSlice)})
	require.NoError(t, h.Handle(context.Background(), makeRecord(logutil.LevelInfo, "m")))
	require.Equal(t, "0 validation errors", decodeJSON(t, buf.Bytes())["errs"])
}

// Test_zerologHandler_inlinedGroupNesting covers an inlined (empty-key) group appearing below the
// root, where it is flattened onto its enclosing group rather than onto the record: inside a named
// group, and inside an open WithGroup. The root case is covered by the trace_id suites.
func Test_zerologHandler_inlinedGroupNesting(t *testing.T) {
	t.Parallel()

	// An inlined group nested inside a named group flattens onto that group.
	buf := &bytes.Buffer{}
	rec := makeRecord(logutil.LevelInfo, "m", grp("g", grp("", slog.String("inlined", "yes")), slog.String("k", "v")))
	require.NoError(t, newLeaf(buf).Handle(context.Background(), rec))

	g, ok := decodeJSON(t, buf.Bytes())["g"].(map[string]any)
	require.True(t, ok, "the named group must be present")
	require.Equal(t, "yes", g["inlined"], "an inlined group nested in a named group flattens onto it")
	require.Equal(t, "v", g["k"])

	// The same, under an open group (the per-record group dict path).
	buf.Reset()
	h := newLeaf(buf).WithGroup("open")
	require.NoError(t, h.Handle(context.Background(), makeRecord(logutil.LevelInfo, "m", grp("", slog.String("inlined", "yes")))))

	open, ok := decodeJSON(t, buf.Bytes())["open"].(map[string]any)
	require.True(t, ok, "the open group must be present")
	require.Equal(t, "yes", open["inlined"])

	// An inlined group holding nothing writes nothing, so its enclosing group is still elided.
	buf.Reset()
	require.NoError(t, newLeaf(buf).Handle(context.Background(), makeRecord(logutil.LevelInfo, "m", grp("g", grp("")))))
	require.NotContains(t, decodeJSON(t, buf.Bytes()), "g", "a group holding only an empty inlined group must be elided")
}

// panicValue panics while rendering itself, after the value has been resolved.
type panicValue struct{}

func (panicValue) MarshalJSON() ([]byte, error) { panic("marshal boom") }

// panicError panics from Error(), the classic hand-written-error failure.
type panicError struct{}

func (panicError) Error() string { panic("Error() boom") }

// panicObject panics from MarshalZerologObject, having already written part of the object. zerolog
// opens the object on the event's buffer before calling the marshaler, so an unguarded recover leaves
// a brace that is never closed — the field the sentinel is written into, and every field after it,
// would be swallowed into it.
type panicObject struct{}

func (panicObject) MarshalZerologObject(e *zerolog.Event) {
	e.Str("partial", "written")

	panic("object boom")
}

// panicObjectError is an error whose zerolog rendering is an object (zerolog's AnErr prefers
// LogObjectMarshaler over Error()), and whose marshaler panics.
type panicObjectError struct{}

func (panicObjectError) Error() string { return "unused: the object rendering wins" }

func (panicObjectError) MarshalZerologObject(e *zerolog.Event) { panic("object error boom") }

// logObject renders as a zerolog object rather than through reflection.
type logObject struct{ code string }

func (o logObject) MarshalZerologObject(e *zerolog.Event) { e.Str("code", o.code) }

// Test_zerologHandler_logObjectMarshalerRendersAsObject pins that a zerolog.LogObjectMarshaler is
// still written as a JSON object on every path, now that it is rendered through a detached dictionary
// rather than by zerolog's Event.Object (see addObjectEvent).
func Test_zerologHandler_logObjectMarshalerRendersAsObject(t *testing.T) {
	t.Parallel()

	want := map[string]any{"code": "E1"}

	// Record-attribute path.
	buf := &bytes.Buffer{}
	require.NoError(t, newLeaf(buf).Handle(context.Background(),
		makeRecord(logutil.LevelInfo, "m", slog.Any("v", logObject{code: "E1"}), slog.String("after", "x"))))

	got := decodeJSON(t, buf.Bytes())
	require.Equal(t, want, got["v"])
	require.Equal(t, "x", got["after"], "the attribute after the object must still be written")

	// Baked (WithAttrs) path.
	buf.Reset()

	h := newLeaf(buf).WithAttrs([]slog.Attr{slog.Any("v", logObject{code: "E2"})})
	require.NoError(t, h.Handle(context.Background(), makeRecord(logutil.LevelInfo, "m")))
	require.Equal(t, map[string]any{"code": "E2"}, decodeJSON(t, buf.Bytes())["v"])

	// An error that is also a LogObjectMarshaler: zerolog's AnErr renders it as an object, not as its
	// message, and so must this handler.
	buf.Reset()
	require.NoError(t, newLeaf(buf).Handle(context.Background(),
		makeRecord(logutil.LevelInfo, "m", slog.Any("err", objectError{code: "E3"}))))
	require.Equal(t, map[string]any{"code": "E3"}, decodeJSON(t, buf.Bytes())["err"])
}

// objectError is an error whose zerolog rendering is an object.
type objectError struct{ code string }

func (objectError) Error() string { return "unused: the object rendering wins" }

func (o objectError) MarshalZerologObject(e *zerolog.Event) { e.Str("code", o.code) }

// Test_zerologHandler_panicWhileRenderingRecovered pins that a value which panics while rendering
// itself cannot take the process down: a log call is what runs on the failure path. slog's own
// handlers recover and write the "!PANIC" sentinel; so does this one, and it counts as a written
// field so an enclosing group is not elided by the failure.
//
// Every assertion decodes the line, so it also pins that the recovery leaves valid JSON behind. That
// is the whole difficulty for a panicking MarshalZerologObject: zerolog writes the key and the opening
// brace before it hands control to the marshaler, so recovering after the fact cannot roll the partial
// object back — the sentinel, the message and the trace ID would all be written inside an object that
// is never closed. Rendering the object separately and attaching it only on success is what keeps the
// failure local (see addObjectEvent), and a panic turning a loud crash into a silently unparseable
// line is the outcome the guard exists to prevent.
func Test_zerologHandler_panicWhileRenderingRecovered(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		attr slog.Attr
		want string
	}{
		{name: "panicking MarshalJSON", attr: slog.Any("v", panicValue{}), want: "!PANIC: marshal boom"},
		{name: "panicking Error", attr: slog.Any("v", panicError{}), want: "!PANIC: Error() boom"},
		{name: "panicking MarshalZerologObject", attr: slog.Any("v", panicObject{}), want: "!PANIC: object boom"},
		{
			name: "panicking MarshalZerologObject on an error",
			attr: slog.Any("v", panicObjectError{}),
			want: "!PANIC: object error boom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Record-attribute path. The attribute after the panicking one must survive: a partially
			// written object would swallow it.
			buf := &bytes.Buffer{}

			require.NotPanics(t, func() {
				require.NoError(t, newLeaf(buf).Handle(context.Background(),
					makeRecord(logutil.LevelInfo, "m", tt.attr, slog.String("after", "x"))))
			}, "a panic while rendering a value must not escape the log call")

			got := decodeJSON(t, buf.Bytes())
			require.Equal(t, tt.want, got["v"])
			require.Equal(t, "x", got["after"], "the attribute after the panicking one must still be written")
			require.Equal(t, "m", got["message"], "the message must not be swallowed by a half-written value")

			// Baked (WithAttrs) path: the corruption would land in the zerolog context and be replayed
			// on every subsequent record of the derived logger, so both records are checked.
			buf.Reset()

			var h slog.Handler

			require.NotPanics(t, func() { h = newLeaf(buf).WithAttrs([]slog.Attr{tt.attr}) })

			for i := range 2 {
				buf.Reset()
				require.NoError(t, h.Handle(context.Background(), makeRecord(logutil.LevelInfo, "m")))
				require.Equal(t, tt.want, decodeJSON(t, buf.Bytes())["v"], "record %d", i)
			}

			// The sentinel is a written field, so its group survives rather than being elided.
			buf.Reset()
			require.NoError(t, newLeaf(buf).Handle(context.Background(),
				makeRecord(logutil.LevelInfo, "m", grp("g", tt.attr))))

			g, ok := decodeJSON(t, buf.Bytes())["g"].(map[string]any)
			require.True(t, ok, "the group must survive: the sentinel is a field")
			require.Equal(t, tt.want, g["v"])
		})
	}
}

// Test_zerologHandler_timeAndDurationAttrs pins that time- and duration-valued attributes are
// encoded like the standard library's handler (a nanosecond count, and RFC 3339 with nanosecond
// precision) rather than through zerolog's process-global DurationFieldUnit/TimeFieldFormat, whose
// defaults would render the same slog.Attr in milliseconds and truncate the timestamp — making the
// two backends of this library disagree about the same record.
func Test_zerologHandler_timeAndDurationAttrs(t *testing.T) {
	t.Parallel()

	ts := time.Date(2026, 7, 12, 10, 30, 0, 123456789, time.UTC)

	buf := &bytes.Buffer{}
	rec := makeRecord(logutil.LevelInfo, "m",
		slog.Duration("latency", 1500*time.Millisecond),
		slog.Duration("tiny", 250*time.Nanosecond),
		slog.Time("ts", ts),
	)
	require.NoError(t, newLeaf(buf).Handle(context.Background(), rec))

	m := decodeJSON(t, buf.Bytes())
	require.InDelta(t, float64(1500*time.Millisecond), m["latency"], 1, "a duration is a nanosecond count")
	require.InDelta(t, 250, m["tiny"], 1, "a sub-millisecond duration is not rounded away")
	require.Equal(t, "2026-07-12T10:30:00.123456789Z", m["ts"], "a time attribute keeps nanosecond precision")

	// The baked (WithAttrs) path encodes them identically.
	buf.Reset()
	h := newLeaf(buf).WithAttrs([]slog.Attr{slog.Duration("latency", 1500*time.Millisecond), slog.Time("ts", ts)})
	require.NoError(t, h.Handle(context.Background(), makeRecord(logutil.LevelInfo, "m")))

	m = decodeJSON(t, buf.Bytes())
	require.InDelta(t, float64(1500*time.Millisecond), m["latency"], 1)
	require.Equal(t, "2026-07-12T10:30:00.123456789Z", m["ts"])
}

// Test_zerologHandler_timestampSubSecond pins the record timestamp to RFC 3339 with nanosecond
// precision. It is written explicitly rather than through zerolog's Event.Time, so it keeps its
// sub-second resolution instead of being truncated to a whole second by the process-global
// zerolog.TimeFieldFormat (which defaults to plain RFC 3339, a layout with no fractional seconds).
func Test_zerologHandler_timestampSubSecond(t *testing.T) {
	t.Parallel()

	ts := time.Date(2026, 7, 12, 17, 58, 36, 490566713, time.UTC)

	buf := &bytes.Buffer{}
	require.NoError(t, newLeaf(buf).Handle(context.Background(), slog.NewRecord(ts, logutil.LevelInfo, "m", 0)))
	require.Equal(t, "2026-07-12T17:58:36.490566713Z", decodeJSON(t, buf.Bytes())["time"],
		"the record timestamp must keep nanosecond precision")

	// A non-UTC instant keeps its zone offset.
	buf.Reset()
	require.NoError(t, newLeaf(buf).Handle(context.Background(),
		slog.NewRecord(ts.In(time.FixedZone("CET", 3600)), logutil.LevelInfo, "m", 0)))
	require.Equal(t, "2026-07-12T18:58:36.490566713+01:00", decodeJSON(t, buf.Bytes())["time"])

	// A whole-second instant omits the empty fraction, matching time.RFC3339Nano.
	buf.Reset()
	require.NoError(t, newLeaf(buf).Handle(context.Background(),
		slog.NewRecord(time.Date(2026, 7, 12, 17, 58, 36, 0, time.UTC), logutil.LevelInfo, "m", 0)))
	require.Equal(t, "2026-07-12T17:58:36Z", decodeJSON(t, buf.Bytes())["time"])
}

// plainError is a plain error used to exercise the ErrorMarshalFunc dispatch.
type plainError struct{ s string }

func (e *plainError) Error() string { return e.s }

// withErrorMarshalFunc installs a process-global zerolog.ErrorMarshalFunc for the duration of the
// test and restores it afterwards. The hook is a global, so the tests using it cannot run in parallel.
func withErrorMarshalFunc(t *testing.T, f func(error) any) {
	t.Helper()

	prev := zerolog.ErrorMarshalFunc

	t.Cleanup(func() { zerolog.ErrorMarshalFunc = prev }) //nolint:reassign // the hook is a documented zerolog global; restored after the test.

	zerolog.ErrorMarshalFunc = f //nolint:reassign // ditto: exercising the hook is the point of the test.
}

// Test_zerologHandler_errorMarshalFuncCalledOnce pins that the process-global ErrorMarshalFunc is
// invoked exactly once per error attribute, and that its FIRST result is what reaches the wire.
//
// Asking the hook what it would return and then letting zerolog's AnErr call it again runs it twice:
// it doubles the cost of a hook that captures a stack or emits a metric — the usual reasons to replace
// it — and a stateful hook would render the second result rather than the first, which is a different
// value.
func Test_zerologHandler_errorMarshalFuncCalledOnce(t *testing.T) { //nolint:paralleltest // mutates a zerolog global.
	tests := []struct {
		name string
		log  func(h slog.Handler) slog.Record
	}{
		{
			name: "record attribute",
			log: func(_ slog.Handler) slog.Record {
				return makeRecord(logutil.LevelInfo, "m", slog.Any("error", &plainError{"boom"}))
			},
		},
		{
			name: "inside a group",
			log: func(_ slog.Handler) slog.Record {
				return makeRecord(logutil.LevelInfo, "m", grp("g", slog.Any("error", &plainError{"boom"})))
			},
		},
	}

	for _, tt := range tests { //nolint:paralleltest // mutates a zerolog global.
		t.Run(tt.name, func(t *testing.T) {
			var calls atomic.Int64

			withErrorMarshalFunc(t, func(err error) any {
				n := calls.Add(1)

				return fmt.Errorf("%w #%d", err, n)
			})

			buf := &bytes.Buffer{}
			h := newLeaf(buf)

			require.NoError(t, h.Handle(context.Background(), tt.log(h)))
			require.Equal(t, int64(1), calls.Load(), "the hook must be invoked exactly once: %s", buf)
			require.Contains(t, buf.String(), "boom #1", "the hook's first result must be the one rendered")
		})
	}

	t.Run("baked WithAttrs", func(t *testing.T) {
		var calls atomic.Int64

		withErrorMarshalFunc(t, func(err error) any {
			n := calls.Add(1)

			return fmt.Errorf("%w #%d", err, n)
		})

		buf := &bytes.Buffer{}
		h := newLeaf(buf).WithAttrs([]slog.Attr{slog.Any("error", &plainError{"boom"})})

		require.Equal(t, int64(1), calls.Load(), "baking the attribute must invoke the hook exactly once")
		require.NoError(t, h.Handle(context.Background(), makeRecord(logutil.LevelInfo, "m")))
		require.Contains(t, buf.String(), "boom #1", "the hook's first result must be the one baked")
	})
}

// Test_zerologHandler_errorMarshalFuncResultShapes pins that every shape zerolog's AnErr accepts back
// from the hook is handled identically here — including nil, which writes NO field, so an enclosing
// group left with nothing else is elided rather than rendered as a bare "{}".
func Test_zerologHandler_errorMarshalFuncResultShapes(t *testing.T) { //nolint:paralleltest // mutates a zerolog global.
	tests := []struct {
		name      string
		result    func(error) any
		wantValue any
		wantGroup bool // whether a group holding only this error survives
	}{
		{
			name:      "nil: no field at all",
			result:    func(error) any { return nil },
			wantValue: nil,
			wantGroup: false,
		},
		{
			name:      "a string",
			result:    func(error) any { return "as-a-string" },
			wantValue: "as-a-string",
			wantGroup: true,
		},
		{
			name:      "another error",
			result:    func(error) any { return errors.New("replaced") },
			wantValue: "replaced",
			wantGroup: true,
		},
		{
			name:      "an arbitrary value, rendered by reflection",
			result:    func(error) any { return map[string]any{"code": "E1"} },
			wantValue: map[string]any{"code": "E1"},
			wantGroup: true,
		},
		{
			name:      "a LogObjectMarshaler, rendered as an object",
			result:    func(error) any { return logObject{code: "E2"} },
			wantValue: map[string]any{"code": "E2"},
			wantGroup: true,
		},
		{
			name:      "a typed-nil error: no field",
			result:    func(error) any { return (*plainError)(nil) },
			wantValue: nil,
			wantGroup: false,
		},
	}

	for _, tt := range tests { //nolint:paralleltest // mutates a zerolog global.
		t.Run(tt.name, func(t *testing.T) {
			withErrorMarshalFunc(t, tt.result)

			// Record path.
			buf := &bytes.Buffer{}
			require.NoError(t, newLeaf(buf).Handle(context.Background(),
				makeRecord(logutil.LevelInfo, "m", slog.Any("err", &plainError{"original"}))))

			assertErrField(t, decodeJSON(t, buf.Bytes()), tt.wantValue, "the record path")

			// Baked path must agree. It is asserted separately, and by absence rather than by a nil
			// value, because a key that is missing and a key whose value is null both decode to nil:
			// asserting equality alone would let a hook result of nil write "err":null here — writing a
			// field where zerolog writes none, and, under the reserved key, replacing the trace ID.
			buf.Reset()

			h := newLeaf(buf).WithAttrs([]slog.Attr{slog.Any("err", &plainError{"original"})})
			require.NoError(t, h.Handle(context.Background(), makeRecord(logutil.LevelInfo, "m")))
			assertErrField(t, decodeJSON(t, buf.Bytes()), tt.wantValue, "the baked path")

			// A group holding only this error is elided exactly when the error writes no field: a
			// reported-but-unwritten field would leave a bare "{}" behind.
			buf.Reset()
			require.NoError(t, newLeaf(buf).Handle(context.Background(),
				makeRecord(logutil.LevelInfo, "m", grp("g", slog.Any("err", &plainError{"original"})))))

			if tt.wantGroup {
				require.Contains(t, decodeJSON(t, buf.Bytes()), "g")
			} else {
				require.NotContains(t, decodeJSON(t, buf.Bytes()), "g",
					"an error that writes no field must not leave a bare {} behind")
			}
		})
	}
}

// nilGuardObjectError is a typed nil that is BOTH an error and a zerolog.LogObjectMarshaler guarding
// its nil receiver, so zerolog renders it as an object rather than dropping it. This backend drops it
// anyway — see Test_zerologHandler_typedNilObjectErrorOmitted.
type nilGuardObjectError struct{ s string }

func (e *nilGuardObjectError) Error() string { return "unused" }

func (e *nilGuardObjectError) MarshalZerologObject(ev *zerolog.Event) {
	if e == nil {
		ev.Str("nil", "receiver")

		return
	}

	ev.Str("s", e.s)
}

// Test_zerologHandler_typedNilObjectErrorOmitted pins that the typed-nil test PRECEDES the hook and the
// LogObjectMarshaler check, so every typed-nil error writes no field — even one that can render itself
// as an object, which raw zerolog (asserted here as the reference) does write.
//
// This is a deliberate divergence from zerolog, and the reason is the sibling backend: logutil answers
// the same "does this attribute write a field?" question with a filter that can see neither
// zerolog.LogObjectMarshaler nor zerolog.ErrorMarshalFunc, so a rule conditional on either cannot be
// mirrored there. Ordering the nil test after them made the two backends disagree — logutil dropped the
// field and its enclosing group where this one wrote them, and under the reserved key the two
// correlated the same record under different trace IDs.
func Test_zerologHandler_typedNilObjectErrorOmitted(t *testing.T) {
	t.Parallel()

	var typedNil *nilGuardObjectError

	// What raw zerolog does with the same value, as the reference: it writes the object.
	ref := &bytes.Buffer{}
	zl := zerolog.New(ref)
	zl.Log().AnErr("err", typedNil).Msg("m")

	require.Contains(t, ref.String(), `"err":{"nil":"receiver"}`, "reference: zerolog renders it as an object")

	// Record path.
	buf := &bytes.Buffer{}
	require.NoError(t, newLeaf(buf).Handle(context.Background(),
		makeRecord(logutil.LevelInfo, "m", slog.Any("err", typedNil))))
	require.NotContains(t, decodeJSON(t, buf.Bytes()), "err",
		"a typed-nil error must write no field, whatever it could render itself into")

	// The baked path must agree.
	buf.Reset()

	h := newLeaf(buf).WithAttrs([]slog.Attr{slog.Any("err", typedNil)})
	require.NoError(t, h.Handle(context.Background(), makeRecord(logutil.LevelInfo, "m")))
	require.NotContains(t, decodeJSON(t, buf.Bytes()), "err", "the baked path must agree")

	// A group left with no other fields by it is elided, on both paths.
	buf.Reset()
	require.NoError(t, newLeaf(buf).Handle(context.Background(),
		makeRecord(logutil.LevelInfo, "m", grp("cause", slog.Any("err", typedNil)))))
	require.NotContains(t, decodeJSON(t, buf.Bytes()), "cause", "the enclosing group must be elided with it")
}

// Test_zerologHandler_typedNilErrorHookNotInvoked pins the corollary: the typed-nil test precedes the
// hook, so zerolog.ErrorMarshalFunc is not invoked for one. A hook that renders a typed nil into
// something visible would otherwise make this backend write a field logutil's cannot, since logutil
// cannot see the hook at all.
func Test_zerologHandler_typedNilErrorHookNotInvoked(t *testing.T) { //nolint:paralleltest // mutates a zerolog global.
	var calls atomic.Int64

	withErrorMarshalFunc(t, func(error) any {
		calls.Add(1)

		return "RENDERED-BY-HOOK"
	})

	var typedNil *plainError

	buf := &bytes.Buffer{}
	require.NoError(t, newLeaf(buf).Handle(context.Background(),
		makeRecord(logutil.LevelInfo, "m", slog.Any("err", typedNil))))

	require.NotContains(t, decodeJSON(t, buf.Bytes()), "err", "a typed nil must write no field")
	require.Zero(t, calls.Load(), "the hook must not be invoked for a typed-nil error")

	// The baked path must agree.
	buf.Reset()

	h := newLeaf(buf).WithAttrs([]slog.Attr{slog.Any("err", typedNil)})
	require.NoError(t, h.Handle(context.Background(), makeRecord(logutil.LevelInfo, "m")))
	require.NotContains(t, decodeJSON(t, buf.Bytes()), "err", "the baked path must agree")
	require.Zero(t, calls.Load(), "the baked path must not invoke the hook either")
}

// Test_zerologHandler_emptySourceElided pins that an empty *slog.Source writes no field, matching
// slog's own handlers and the sibling logutil backend. Under the reserved trace ID key it would
// otherwise be read as a caller-supplied trace ID and suppress the injected one, leaving the record
// with a null correlation ID.
func Test_zerologHandler_emptySourceElided(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		src  *slog.Source
	}{
		{name: "nil Source", src: nil},
		{name: "zero-valued Source", src: &slog.Source{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Under the reserved key it must not suppress the injected trace ID.
			buf := &bytes.Buffer{}
			h := newLeaf(buf)
			h.traceIDFn = func() string { return "TID" }

			require.NoError(t, h.Handle(context.Background(),
				makeRecord(logutil.LevelInfo, "m", slog.Any(logutil.TraceIDKey, tt.src))))
			require.Equal(t, "TID", decodeJSON(t, buf.Bytes())[logutil.TraceIDKey],
				"an empty Source writes nothing, so it cannot stand in for the trace ID")

			// A group holding only it is elided.
			buf.Reset()
			require.NoError(t, newLeaf(buf).Handle(context.Background(),
				makeRecord(logutil.LevelInfo, "m", grp("g", slog.Any("s", tt.src)))))
			require.NotContains(t, decodeJSON(t, buf.Bytes()), "g")

			// Baked, it writes nothing either.
			buf.Reset()

			baked := newLeaf(buf).WithAttrs([]slog.Attr{slog.Any("s", tt.src)})
			require.NoError(t, baked.Handle(context.Background(), makeRecord(logutil.LevelInfo, "m")))
			require.NotContains(t, decodeJSON(t, buf.Bytes()), "s")
		})
	}
}

// Test_zerologHandler_nonEmptySourceRendered guards the other side of the rule: a real caller location
// is a field like any other.
func Test_zerologHandler_nonEmptySourceRendered(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	require.NoError(t, newLeaf(buf).Handle(context.Background(), makeRecord(logutil.LevelInfo, "m",
		slog.Any("caller", &slog.Source{Function: "f", File: "f.go", Line: 12}))))
	require.Contains(t, decodeJSON(t, buf.Bytes()), "caller")
}

// Test_addObjectEvent_recyclesTheAbandonedDict pins that a panicking marshaler's half-written
// dictionary is returned to zerolog's Event pool rather than leaked.
//
// It is asserted by allocation count because a leak has no other visible symptom: the line stays valid
// and the sentinel is still written, so nothing fails — the pool simply has to allocate a fresh Event
// for every panicking record forever. Dropping the d.Send() call takes this from 1 alloc/op to 3.
func Test_addObjectEvent_recyclesTheAbandonedDict(t *testing.T) { //nolint:paralleltest // AllocsPerRun cannot run in a parallel test.
	tests := []struct {
		name  string
		build func(w io.Writer) slog.Handler
		rec   slog.Record
	}{
		{
			name:  "record attribute",
			build: func(w io.Writer) slog.Handler { return newLeaf(w) },
			rec:   makeRecord(logutil.LevelInfo, "m", slog.Any("v", panicObject{})),
		},
		{
			name: "baked attribute, replayed per record",
			build: func(w io.Writer) slog.Handler {
				return newLeaf(w).WithAttrs([]slog.Attr{slog.Any("v", logObject{code: "E"})})
			},
			rec: makeRecord(logutil.LevelInfo, "m", slog.Any("v", panicObject{})),
		},
	}

	for _, tt := range tests { //nolint:paralleltest // AllocsPerRun cannot run in a parallel test.
		t.Run(tt.name, func(t *testing.T) {
			h := tt.build(io.Discard)

			allocs := testing.AllocsPerRun(500, func() {
				_ = h.Handle(context.Background(), tt.rec)
			})

			require.LessOrEqual(t, allocs, 2.0,
				"the abandoned dictionary must be recycled into zerolog's pool, not leaked (got %.0f allocs/op)", allocs)
		})
	}
}

// Test_applyObjectContext_recyclesTheAbandonedDict is the context-path counterpart: a panic inside a
// marshaler baked with WithAttrs must not leak the pooled Event either. Here the leak is the whole
// reason the function exists — zerolog's own Context.Object renders into a scratch event it never
// returns to the pool on a panic — so it is the one behavior that must be pinned.
func Test_applyObjectContext_recyclesTheAbandonedDict(t *testing.T) { //nolint:paralleltest // AllocsPerRun cannot run in a parallel test.
	allocs := testing.AllocsPerRun(500, func() {
		_ = newLeaf(io.Discard).WithAttrs([]slog.Attr{slog.Any("v", panicObject{})})
	})

	// A derivation allocates the handler itself and the zerolog context (4 allocs, 5 under -race); a
	// leaked pooled dictionary shows up as 2 more, since the pool must build a fresh Event per panic.
	require.LessOrEqual(t, allocs, 5.0,
		"the abandoned dictionary must be recycled into zerolog's pool, not leaked (got %.0f allocs/op)", allocs)
}

// assertErrField asserts what the "err" key holds, distinguishing an absent key from one whose value is
// null: both decode to nil, so require.Equal(nil, m["err"]) cannot tell "no field was written" from
// "a null field was written". The difference is the whole point of reporting what was really written —
// a null where zerolog writes nothing leaves a bare "{}" behind an enclosing group, and replaces the
// trace ID when the key is the reserved one.
func assertErrField(t *testing.T, m map[string]any, want any, path string) {
	t.Helper()

	if want == nil {
		require.NotContains(t, m, "err", "%s must write no field at all", path)

		return
	}

	require.Equal(t, want, m["err"], "%s", path)
}

// hookedObjectError is BOTH an error and a zerolog.LogObjectMarshaler, with no nil receiver involved.
// It is the shape that distinguishes the two dispatch orders in addAnyEvent: checking error first sends
// it through zerolog.ErrorMarshalFunc (as zerolog's AnErr does), while checking LogObjectMarshaler first
// would render the object directly and never call the hook at all.
type hookedObjectError struct{ s string }

func (e *hookedObjectError) Error() string { return e.s }

func (e *hookedObjectError) MarshalZerologObject(ev *zerolog.Event) { ev.Str("raw", e.s) }

// Test_zerologHandler_errorHookWinsOverObjectMarshaler pins that an error is dispatched as an error
// even when it also implements zerolog.LogObjectMarshaler, so the process-global ErrorMarshalFunc still
// governs it. Checking the marshaler first would bypass the hook entirely — a redacting or
// stack-capturing hook would silently stop firing, and the value it was installed to suppress would ship
// as an object.
func Test_zerologHandler_errorHookWinsOverObjectMarshaler(t *testing.T) { //nolint:paralleltest // mutates a zerolog global.
	var calls atomic.Int64

	withErrorMarshalFunc(t, func(error) any {
		calls.Add(1)

		return "REDACTED-BY-HOOK"
	})

	value := &hookedObjectError{s: "secret"}

	// What raw zerolog does with the same value, as the reference.
	ref := &bytes.Buffer{}
	zl := zerolog.New(ref)
	zl.Log().AnErr("e", value).Msg("m")

	require.Contains(t, ref.String(), `"e":"REDACTED-BY-HOOK"`, "reference: zerolog routes it through the hook")

	for _, tc := range []struct { //nolint:paralleltest // mutates a zerolog global.
		name string
		emit func(w *bytes.Buffer) error
	}{
		{
			name: "record attribute",
			emit: func(w *bytes.Buffer) error {
				return newLeaf(w).Handle(context.Background(), makeRecord(logutil.LevelInfo, "m", slog.Any("e", value)))
			},
		},
		{
			name: "baked attribute",
			emit: func(w *bytes.Buffer) error {
				h := newLeaf(w).WithAttrs([]slog.Attr{slog.Any("e", value)})

				return h.Handle(context.Background(), makeRecord(logutil.LevelInfo, "m"))
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls.Store(0)

			buf := &bytes.Buffer{}
			require.NoError(t, tc.emit(buf))

			require.Equal(t, "REDACTED-BY-HOOK", decodeJSON(t, buf.Bytes())["e"],
				"the hook's result must win over the value's own object rendering")
			require.Equal(t, int64(1), calls.Load(), "the hook must be invoked exactly once")
		})
	}
}

// zeroValueValuer resolves to the ZERO slog.Value. Under an empty key the standard library elides the
// attribute — it resolves first and only then tests for emptiness — so this backend must too. Testing
// for emptiness before resolving cannot see it: a LogValuer is never equal to the zero Value, since its
// kind differs.
type zeroValueValuer struct{}

func (zeroValueValuer) LogValue() slog.Value { return slog.Value{} }

// Test_zerologHandler_valuerResolvingToZeroAttrElided pins that the zero-Attr test follows resolution,
// as slog's appendAttr orders it. Otherwise an empty-key valuer resolving to the zero Value is written
// as a null field under an empty key — and, under a group named for the reserved key, that stray field
// makes the group render, suppressing the injected trace ID and replacing it with {"":null}.
func Test_zerologHandler_valuerResolvingToZeroAttrElided(t *testing.T) {
	t.Parallel()

	emptyKeyZero := slog.Attr{Key: "", Value: slog.AnyValue(zeroValueValuer{})}

	t.Run("at the root it writes no field", func(t *testing.T) {
		t.Parallel()

		buf := &bytes.Buffer{}
		h := newLeaf(buf)
		h.traceIDFn = func() string { return "TID" }

		require.NoError(t, h.Handle(context.Background(), makeRecord(logutil.LevelInfo, "m", emptyKeyZero)))

		got := decodeJSON(t, buf.Bytes())
		require.NotContains(t, got, "", "a value resolving to the zero Attr must write no field")
		require.Equal(t, "TID", got[logutil.TraceIDKey])
	})

	t.Run("it does not stand in for the trace ID", func(t *testing.T) {
		t.Parallel()

		buf := &bytes.Buffer{}
		h := newLeaf(buf)
		h.traceIDFn = func() string { return "TID" }

		grouped := h.WithGroup(logutil.TraceIDKey)
		require.NoError(t, grouped.Handle(context.Background(), makeRecord(logutil.LevelInfo, "m", emptyKeyZero)))

		require.Equal(t, "TID", decodeJSON(t, buf.Bytes())[logutil.TraceIDKey],
			"a group left empty by the value must not suppress the injected trace ID")
	})

	t.Run("baked, it writes no field", func(t *testing.T) {
		t.Parallel()

		buf := &bytes.Buffer{}
		h := newLeaf(buf).WithAttrs([]slog.Attr{emptyKeyZero})

		require.NoError(t, h.Handle(context.Background(), makeRecord(logutil.LevelInfo, "m")))
		require.NotContains(t, decodeJSON(t, buf.Bytes()), "", "the baked path must agree")
	})
}

// Test_zerologHandler_sourceUnderEmptyKeyInlines pins slog's *slog.Source special case: a populated one
// is rendered as its function/file/line group, so an empty key inlines it onto the enclosing level
// rather than nesting it under "". Writing the value through reflection instead happens to produce the
// same object under a named key — slog.Source carries lowercase json tags — so only the empty-key case
// reveals the difference.
func Test_zerologHandler_sourceUnderEmptyKeyInlines(t *testing.T) {
	t.Parallel()

	src := &slog.Source{Function: "pkg.Fn", File: "/x/y.go", Line: 7}
	want := map[string]any{"function": "pkg.Fn", "file": "/x/y.go", "line": 7.0}

	t.Run("an empty key inlines it onto the root", func(t *testing.T) {
		t.Parallel()

		buf := &bytes.Buffer{}
		require.NoError(t, newLeaf(buf).Handle(context.Background(),
			makeRecord(logutil.LevelInfo, "m", slog.Any("", src))))

		got := decodeJSON(t, buf.Bytes())
		require.NotContains(t, got, "", "it must not nest under an empty key")

		for k, v := range want {
			require.Equal(t, v, got[k], "field %q must be inlined onto the root", k)
		}
	})

	t.Run("an empty key inlines it into an enclosing group", func(t *testing.T) {
		t.Parallel()

		buf := &bytes.Buffer{}
		require.NoError(t, newLeaf(buf).Handle(context.Background(),
			makeRecord(logutil.LevelInfo, "m", grp("g", slog.Any("", src)))))

		require.Equal(t, want, decodeJSON(t, buf.Bytes())["g"])
	})

	t.Run("a named key nests it", func(t *testing.T) {
		t.Parallel()

		buf := &bytes.Buffer{}
		require.NoError(t, newLeaf(buf).Handle(context.Background(),
			makeRecord(logutil.LevelInfo, "m", slog.Any("caller", src))))

		require.Equal(t, want, decodeJSON(t, buf.Bytes())["caller"])
	})

	t.Run("baked, an empty key inlines it too", func(t *testing.T) {
		t.Parallel()

		buf := &bytes.Buffer{}
		h := newLeaf(buf).WithAttrs([]slog.Attr{slog.Any("", src)})

		require.NoError(t, h.Handle(context.Background(), makeRecord(logutil.LevelInfo, "m")))

		got := decodeJSON(t, buf.Bytes())
		require.NotContains(t, got, "")
		require.Equal(t, "pkg.Fn", got["function"])
	})
}

// Test_zerologHandler_typedNilObjectMarshalerRendersNull pins that a typed-nil LogObjectMarshaler which
// is NOT an error is written as null rather than run: calling its marshaler would dereference the nil
// receiver, and the "!PANIC" sentinel is a poor rendering of what is simply a nil value. The standard
// library writes null for it, so both backends agree.
//
// An *error* that is a typed nil is not routed here — it goes through zerolog's own AnErr dispatch — so
// one guarding its nil receiver still renders itself as an object (see
// Test_zerologHandler_typedNilObjectErrorStillRendered, which must keep passing).
func Test_zerologHandler_typedNilObjectMarshalerRendersNull(t *testing.T) {
	t.Parallel()

	var typedNil *logObjectPtr

	t.Run("record attribute", func(t *testing.T) {
		t.Parallel()

		buf := &bytes.Buffer{}
		require.NoError(t, newLeaf(buf).Handle(context.Background(),
			makeRecord(logutil.LevelInfo, "m", slog.Any("v", typedNil))))

		got := decodeJSON(t, buf.Bytes())
		require.Contains(t, got, "v", "a nil value must be written, as null")
		require.Nil(t, got["v"])
	})

	t.Run("baked attribute", func(t *testing.T) {
		t.Parallel()

		buf := &bytes.Buffer{}
		h := newLeaf(buf).WithAttrs([]slog.Attr{slog.Any("v", typedNil)})

		require.NoError(t, h.Handle(context.Background(), makeRecord(logutil.LevelInfo, "m")))

		got := decodeJSON(t, buf.Bytes())
		require.Contains(t, got, "v")
		require.Nil(t, got["v"])
	})
}

// logObjectPtr is a pointer-receiver LogObjectMarshaler with NO nil-receiver guard: invoking it on a
// typed nil panics.
type logObjectPtr struct{ code string }

func (o *logObjectPtr) MarshalZerologObject(e *zerolog.Event) { e.Str("code", o.code) }

// Test_elidedGroupRecyclesThePooledDict pins the three group paths that take a pooled zerolog Event to
// build a dictionary and then discover the group renders nothing. Each must return that Event to the
// pool: nothing about the output says whether it did, so only an allocation assertion can hold it, and
// a leak costs 2 allocs/op on every record carrying such a group — forever, since the pool must then
// build a fresh Event each time.
func Test_elidedGroupRecyclesThePooledDict(t *testing.T) { //nolint:paralleltest // AllocsPerRun cannot run in a parallel test.
	var e *typedNilError

	tests := []struct {
		name  string
		build func(w io.Writer) slog.Handler
		rec   slog.Record
	}{
		{
			// addGroupEvent: a record-level group whose members all elide.
			name:  "a record group that renders nothing",
			build: func(w io.Writer) slog.Handler { return newLeaf(w) },
			rec:   makeRecord(logutil.LevelInfo, "m", grp("g", slog.Any("err", e))),
		},
		{
			// buildGroupDict: an open group the record leaves empty.
			name:  "an open group the record leaves empty",
			build: func(w io.Writer) slog.Handler { return newLeaf(w).WithGroup("wg") },
			rec:   makeRecord(logutil.LevelInfo, "m"),
		},
	}

	for _, tt := range tests { //nolint:paralleltest // AllocsPerRun cannot run in a parallel test.
		t.Run(tt.name, func(t *testing.T) {
			h := tt.build(io.Discard)

			// zerolog's Event pool is a process-global sync.Pool that the rest of this suite leaves
			// populated, and a leak spends those spares before it has to allocate — which is enough to
			// hide it over a few hundred runs. Draining the pool first (a sync.Pool is cleared by a GC,
			// its victim cache by the second) makes the measurement independent of what ran before.
			runtime.GC()
			runtime.GC()

			allocs := testing.AllocsPerRun(500, func() {
				_ = h.Handle(context.Background(), tt.rec)
			})

			// Handling the record allocates nothing (1 under -race, whose instrumentation adds one);
			// a leaked pooled dictionary shows up as 2 more, since the pool must then build a fresh
			// Event for every such record.
			require.LessOrEqual(t, allocs, 1.0,
				"the unused dictionary must be recycled into zerolog's pool, not leaked (got %.0f allocs/op)", allocs)
		})
	}

	// applyGroupContext: a baked group that renders nothing. Measured on the derivation itself, which is
	// where the dictionary is taken and discarded.
	t.Run("a baked group that renders nothing", func(t *testing.T) {
		allocs := testing.AllocsPerRun(500, func() {
			_ = newLeaf(io.Discard).WithAttrs([]slog.Attr{grp("g", slog.Any("err", e))})
		})

		// A derivation allocates the handler and the zerolog context; a leaked dictionary adds 2 more.
		require.LessOrEqual(t, allocs, 4.0,
			"the unused dictionary must be recycled into zerolog's pool, not leaked (got %.0f allocs/op)", allocs)
	})
}

// Test_timeBufSize_keepsTheWorstCaseTimestampOnTheStack pins the one constant every record depends on:
// the timestamp is formatted into a stack buffer of timeBufSize bytes, and a value that does not fit
// makes AppendFormat grow the slice on the heap — silently costing an allocation on EVERY record, which
// is the whole "zero allocations" property of this backend. The output stays correct either way, so no
// output assertion can hold it.
//
// The record carries time.Time's widest rendering (a negative twelve-digit year with a numeric zone
// offset, 46 bytes) both as the record timestamp and as an attribute, so both call sites are covered.
func Test_timeBufSize_keepsTheWorstCaseTimestampOnTheStack(t *testing.T) { //nolint:paralleltest // AllocsPerRun cannot run in a parallel test.
	worst := time.Date(-292277022399, 1, 1, 0, 0, 0, 999999999, time.FixedZone("w", -12*3600-59*60))

	rec := slog.NewRecord(worst, logutil.LevelInfo, "m", 0)
	rec.AddAttrs(slog.Time("t", worst))

	h := newLeaf(io.Discard)

	allocs := testing.AllocsPerRun(500, func() {
		_ = h.Handle(context.Background(), rec)
	})

	// A buffer too small to hold the value makes AppendFormat grow it on the heap, once for the record
	// timestamp and once for the attribute (1 alloc/op is the -race instrumentation's own).
	require.LessOrEqual(t, allocs, 1.0,
		"timeBufSize must cover the widest timestamp so formatting stays on the stack (got %.0f allocs/op)", allocs)
}

// Test_sourceDictOf_matchesTheAttributeRule pins the caller-location rule against its sibling. The same
// question — which components of a caller location get written — is answered in two places: sourceAttrs,
// for a *slog.Source logged as an attribute, and sourceDictOf, for the record's own location. They have
// already drifted apart once. Nothing in the output of an ordinary record can catch that, because a
// resolvable PC sets all three components, so this drives the rule directly, over every combination.
func Test_sourceDictOf_matchesTheAttributeRule(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		src  slog.Source
		want map[string]any // nil means: write no field at all
	}{
		{name: "empty", src: slog.Source{}, want: nil},
		{name: "function only", src: slog.Source{Function: "pkg.Fn"}, want: map[string]any{"function": "pkg.Fn"}},
		{name: "file only", src: slog.Source{File: "f.go"}, want: map[string]any{"file": "f.go"}},
		{name: "line only", src: slog.Source{Line: 7}, want: map[string]any{"line": 7.0}},
		{
			name: "function and file",
			src:  slog.Source{Function: "pkg.Fn", File: "f.go"},
			want: map[string]any{"function": "pkg.Fn", "file": "f.go"},
		},
		{
			name: "file and line",
			src:  slog.Source{File: "f.go", Line: 7},
			want: map[string]any{"file": "f.go", "line": 7.0},
		},
		{
			name: "function and line",
			src:  slog.Source{Function: "pkg.Fn", Line: 7},
			want: map[string]any{"function": "pkg.Fn", "line": 7.0},
		},
		{
			name: "all three",
			src:  slog.Source{Function: "pkg.Fn", File: "f.go", Line: 7},
			want: map[string]any{"function": "pkg.Fn", "file": "f.go", "line": 7.0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			buf := &bytes.Buffer{}

			zl := zerolog.New(buf)
			e := zl.Log()

			if d, ok := sourceDictOf(tt.src); ok {
				e.Dict("source", d)
			}

			e.Msg("m")

			got := decodeJSON(t, buf.Bytes())

			if tt.want == nil {
				require.NotContains(t, got, "source", "an empty caller location must write no field")

				return
			}

			require.Equal(t, tt.want, got["source"])

			// And the attribute path must answer identically: a zero component is omitted by both, so
			// the record's location and a *slog.Source attribute can never render differently.
			attrs := &bytes.Buffer{}

			zla := zerolog.New(attrs)
			ea := zla.Log()
			require.True(t, addGroupEvent(ea, "source", sourceAttrs(&tt.src)))
			ea.Msg("m")

			require.Equal(t, got["source"], decodeJSON(t, attrs.Bytes())["source"],
				"sourceDictOf and sourceAttrs must write the same components")
		})
	}
}
