package logsrv_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"maps"
	"runtime"
	"slices"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"github.com/tecnickcom/nurago/pkg/logsrv"
	"github.com/tecnickcom/nurago/pkg/logutil"
)

// This file is the only place the two backends can be compared: logutil cannot import logsrv (logsrv
// imports it), and logutil's own tests cannot construct a zerolog value at all. Every defect this suite
// exists to catch has been of one kind — a rule about whether an attribute writes a field, applied in
// one backend and not the other — and neither package's tests could see it alone.

// nilObjectError is a typed nil that is BOTH an error and a zerolog.LogObjectMarshaler guarding its nil
// receiver. zerolog's AnErr writes it as an object; logutil's filter cannot even see the interface. It
// is the shape on which the two backends' "writes no field" rules must agree without either being able
// to consult the other's world.
type nilObjectError struct{ s string }

func (e *nilObjectError) Error() string { return e.s }

func (e *nilObjectError) MarshalZerologObject(ev *zerolog.Event) {
	if e == nil {
		ev.Str("nil", "receiver")

		return
	}

	ev.Str("s", e.s)
}

// sliceError is an aggregate error whose underlying kind is a slice (the shape of
// validator.ValidationErrors). A nil one is NOT a nil pointer, so both backends must still render it.
type sliceError []string

func (s sliceError) Error() string { return "validation" }

// backendOpt configures the two handlers a case is driven through. The defaults (JSON, no caller
// location) are not enough on their own: a rule can be applied to an attribute and not to the record's
// own caller location, which is exactly how the two backends last drifted apart, so Source must be an
// axis of this test rather than a setting it happens not to exercise.
type backendOpt func(*logutil.Config)

// withSource turns on the caller-location field, whose components follow the same write-or-elide rules
// as a *slog.Source attribute — and are produced by different code.
func withSource(c *logutil.Config) { c.Source = true }

// withCommonAttr bakes the attributes into the handler, the path that poisons every line if it diverges.
func withCommonAttr(attrs ...slog.Attr) backendOpt {
	return func(c *logutil.Config) { c.CommonAttr = attrs }
}

// backendLines logs the same record through both backends and returns the two decoded JSON objects.
func backendLines(t *testing.T, log func(*slog.Logger), opts ...backendOpt) (map[string]any, map[string]any) {
	t.Helper()

	out := make([]map[string]any, 2)

	for i, build := range []func(*logutil.Config) slog.Handler{
		logsrv.NewHandler,
		func(c *logutil.Config) slog.Handler { return c.SlogHandler() },
	} {
		var buf bytes.Buffer

		cfg := logutil.DefaultConfig()
		cfg.Out = &buf
		cfg.TraceIDFn = func() string { return "TID" }

		for _, opt := range opts {
			opt(cfg)
		}

		log(slog.New(build(cfg)))

		line := bytes.TrimSpace(buf.Bytes())
		require.True(t, json.Valid(line), "backend %d must emit valid JSON: %s", i, line)
		require.NoError(t, json.Unmarshal(line, &out[i]))
	}

	return out[0], out[1]
}

// fieldNames returns the decoded keys, minus the only two the backends name differently by design: the
// message field, which zerolog calls "message" and slog "msg".
//
// Nothing else is excluded. "time" in particular is NOT — its *value* differs (each backend stamps its
// own time.Now()), but its presence must not, and excluding the key would blind this test to the very
// class of defect it exists to catch.
func fieldNames(m map[string]any) []string {
	keys := slices.Collect(maps.Keys(m))
	keys = slices.DeleteFunc(keys, func(k string) bool {
		return k == "message" || k == "msg"
	})

	slices.Sort(keys)

	return keys
}

// TestBackends_AgreeOnWhatWritesAField pins the invariant the whole trace-ID and group-elision design
// rests on: for any attribute, both backends must agree on whether it writes a field. When they do not,
// the same slog.Attr yields a different field set on each — and under the reserved key, a different
// trace ID, because a field that renders suppresses the injected one.
func TestBackends_AgreeOnWhatWritesAField(t *testing.T) {
	t.Parallel()

	var (
		nilObj   *nilObjectError
		nilPlain = (*typedNilPtrError)(nil)
	)

	tests := []struct {
		name string
		attr slog.Attr
	}{
		{name: "a typed-nil error", attr: slog.Any("err", nilPlain)},
		{name: "a typed-nil error that marshals itself as an object", attr: slog.Any("err", nilObj)},
		{name: "a nil aggregate (slice-kind) error", attr: slog.Any("err", sliceError(nil))},
		{name: "an ordinary error", attr: slog.Any("err", errors.New("boom"))},
		{name: "an empty *slog.Source", attr: slog.Any("err", (*slog.Source)(nil))},
		{name: "a populated *slog.Source", attr: slog.Any("err", &slog.Source{File: "f.go", Line: 7})},
		{name: "a partially populated *slog.Source", attr: slog.Any("err", &slog.Source{Line: 7})},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// As a record attribute, alongside a sibling that must survive either way.
			srv, utl := backendLines(t, func(l *slog.Logger) {
				l.Info("m", tt.attr, slog.Int("a", 1)) //nolint:loggercheck // slog.Attr values, not key-value pairs.
			})
			require.Equal(t, fieldNames(utl), fieldNames(srv), "record attribute: the field sets must match")
			require.Equal(t, utl["err"], srv["err"], "record attribute: the value must match")

			// Inside a group: if the value writes nothing, the group must be elided on BOTH backends.
			srv, utl = backendLines(t, func(l *slog.Logger) {
				l.Info("m", slog.Group("cause", tt.attr), slog.Int("a", 1)) //nolint:loggercheck // ditto.
			})
			require.Equal(t, fieldNames(utl), fieldNames(srv), "in a group: the field sets must match")
			require.Equal(t, utl["cause"], srv["cause"], "in a group: the group must match")

			// Baked, and via the common attributes (which poison every line if they disagree). Values are
			// asserted here too: a field set can match while the two render the value differently.
			srv, utl = backendLines(t, func(l *slog.Logger) {
				l.With(tt.attr).Info("m") //nolint:loggercheck // ditto.
			})
			require.Equal(t, fieldNames(utl), fieldNames(srv), "baked: the field sets must match")
			require.Equal(t, utl["err"], srv["err"], "baked: the value must match")

			srv, utl = backendLines(t, func(l *slog.Logger) { l.Info("m") }, withCommonAttr(tt.attr))
			require.Equal(t, fieldNames(utl), fieldNames(srv), "CommonAttr: the field sets must match")
			require.Equal(t, utl["err"], srv["err"], "CommonAttr: the value must match")

			// With the caller location turned on: the "source" field follows the same write-or-elide
			// rules and is built by different code, so it is its own axis (see backendOpt).
			srv, utl = backendLines(t, func(l *slog.Logger) {
				l.Info("m", tt.attr, slog.Int("a", 1)) //nolint:loggercheck // ditto.
			}, withSource)
			require.Equal(t, fieldNames(utl), fieldNames(srv), "with Source: the field sets must match")
		})
	}
}

// TestBackends_AgreeOnTheTraceIDUnderEveryShape is the consequence of the rule above, at the one key
// where disagreement is worst: a value that writes no field must leave the injected trace ID standing,
// and one that writes a field must replace it — identically on both backends.
func TestBackends_AgreeOnTheTraceIDUnderEveryShape(t *testing.T) {
	t.Parallel()

	var nilObj *nilObjectError

	tests := []struct {
		name string
		val  any
		want any // the root trace_id both backends must ship
	}{
		{name: "a typed-nil error leaves the injected ID", val: (*typedNilPtrError)(nil), want: "TID"},
		{name: "a typed-nil object error leaves the injected ID", val: nilObj, want: "TID"},
		{name: "an empty *slog.Source leaves the injected ID", val: (*slog.Source)(nil), want: "TID"},
		{name: "an ordinary error replaces it", val: errors.New("boom"), want: "boom"},
		{name: "a nil aggregate error replaces it", val: sliceError(nil), want: "validation"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srv, utl := backendLines(t, func(l *slog.Logger) {
				l.Info("m", slog.Any(logutil.TraceIDKey, tt.val))
			})

			require.Equal(t, tt.want, srv[logutil.TraceIDKey], "logsrv")
			require.Equal(t, tt.want, utl[logutil.TraceIDKey], "logutil")
		})
	}
}

// typedNilPtrError is an error on a pointer receiver, so a nil one is a typed nil.
type typedNilPtrError struct{}

func (e *typedNilPtrError) Error() string { return "boom" }

// TestBackends_AgreeOnTheSourceField pins the caller-location field against the same write-or-elide
// rules as a *slog.Source attribute — the two are produced by different code (sourceDict vs
// sourceAttrs), so they can drift, and they did.
//
// slog resolves the PC to a frame and then elides an empty caller location, so a record carrying a PC
// that does not resolve in this binary gets no "source" field. Only a hand-built, replayed or tee'd
// record can carry one — slog.Logger always stamps a live PC — which is exactly the kind of record the
// out-of-range-timestamp repair exists for.
func TestBackends_AgreeOnTheSourceField(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		pc   uintptr
		want bool // whether a "source" field must be written
	}{
		{name: "a live PC writes the caller location", pc: callerPC(), want: true},
		{name: "a zero PC writes nothing", pc: 0, want: false},
		{name: "an unresolvable PC writes nothing", pc: 1, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srv, utl := backendLines(t, func(l *slog.Logger) {
				rec := slog.NewRecord(time.Now(), slog.LevelInfo, "m", tt.pc)
				rec.AddAttrs(slog.Int("a", 1))

				require.NoError(t, l.Handler().Handle(context.Background(), rec))
			}, withSource)

			require.Equal(t, fieldNames(utl), fieldNames(srv), "the field sets must match")

			if tt.want {
				require.Contains(t, srv, "source")
				require.Equal(t, utl["source"], srv["source"], "the caller location must match")

				return
			}

			require.NotContains(t, srv, "source", "an empty caller location must write no field")
			require.NotContains(t, utl, "source")
		})
	}
}

// callerPC returns a real program counter, as slog.Logger stamps into every record.
func callerPC() uintptr {
	var pcs [1]uintptr

	runtime.Callers(1, pcs[:])

	return pcs[0]
}

// TestBackends_AgreeOnTheBuiltInFields pins the presence of the fields neither backend writes as an
// attribute. slog omits the timestamp of a record whose Time is zero; both backends must do the same, or
// a field set differs on a shape no attribute test can reach. It is the case fieldNames stopped
// excluding "time" for.
func TestBackends_AgreeOnTheBuiltInFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		time     time.Time
		wantTime bool
	}{
		{name: "a real timestamp is written", time: time.Now(), wantTime: true},
		{name: "a zero timestamp is omitted", time: time.Time{}, wantTime: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srv, utl := backendLines(t, func(l *slog.Logger) {
				rec := slog.NewRecord(tt.time, slog.LevelInfo, "m", 0)
				rec.AddAttrs(slog.Int("a", 1))

				require.NoError(t, l.Handler().Handle(context.Background(), rec))
			})

			require.Equal(t, fieldNames(utl), fieldNames(srv), "the field sets must match")
			require.Equal(t, tt.wantTime, slices.Contains(fieldNames(srv), "time"))
			require.Contains(t, fieldNames(srv), "level", "the level is always written")
		})
	}
}
