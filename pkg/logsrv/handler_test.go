package logsrv

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"github.com/tecnickcom/gogen/pkg/logutil"
)

// newLeaf builds a bare native leaf handler (no trace/hook wrappers) at TraceLevel so
// every record is emitted, for white-box testing of the encoding paths.
func newLeaf(w io.Writer) *zerologHandler {
	return &zerologHandler{
		logger:   zerolog.New(w).Level(zerolog.TraceLevel),
		minLevel: logutil.LevelTrace,
	}
}

// makeRecord builds a record with a zero time and no caller PC.
func makeRecord(level slog.Level, msg string, attrs ...slog.Attr) slog.Record {
	r := slog.NewRecord(time.Time{}, level, msg, 0)
	r.AddAttrs(attrs...)

	return r
}

// grp builds a group attribute without going through slog.Group's variadic key/value form.
func grp(key string, attrs ...slog.Attr) slog.Attr {
	return slog.Attr{Key: key, Value: slog.GroupValue(attrs...)}
}

func decodeJSON(t *testing.T, b []byte) map[string]any {
	t.Helper()

	m := map[string]any{}
	require.NoError(t, json.Unmarshal(b, &m), "output: %s", b)

	return m
}

// logValuer resolves to a string, exercising the KindLogValuer -> Resolve path.
type logValuer struct{}

func (logValuer) LogValue() slog.Value { return slog.StringValue("resolved") }

func Test_zerologHandler_allKinds(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	h := newLeaf(buf)

	rec := makeRecord(logutil.LevelInfo, "m",
		slog.String("s", "x"),
		slog.Int64("i", -3),
		slog.Uint64("u", 7),
		slog.Float64("f", 1.5),
		slog.Bool("b", true),
		slog.Duration("d", time.Second),
		slog.Time("t", time.Unix(0, 0).UTC()),
		slog.Any("err", errors.New("boom")),
		slog.Any("any", map[string]any{"z": 1}),
		slog.Any("lv", logValuer{}),
		grp("g", slog.String("k", "v")),
		grp("", slog.String("inlined", "yes")),
	)

	require.NoError(t, h.Handle(context.Background(), rec))

	m := decodeJSON(t, buf.Bytes())
	require.Equal(t, "x", m["s"])
	require.InDelta(t, -3, m["i"], 1e-9)
	require.InDelta(t, 7, m["u"], 1e-9)
	require.InEpsilon(t, 1.5, m["f"], 1e-9)
	require.Equal(t, true, m["b"])
	require.Contains(t, m, "d") // duration formatting is a zerolog global; assert presence
	require.Equal(t, "1970-01-01T00:00:00Z", m["t"])
	require.Equal(t, "boom", m["err"]) // error -> AnErr string
	require.Equal(t, map[string]any{"z": float64(1)}, m["any"])
	require.Equal(t, "resolved", m["lv"])              // LogValuer resolved
	require.Equal(t, map[string]any{"k": "v"}, m["g"]) // named group -> nested object
	require.Equal(t, "yes", m["inlined"])              // empty-key group inlined at root
}

func Test_zerologHandler_groups(t *testing.T) {
	t.Parallel()

	// Nested groups produce nested objects. A leading empty attr exercises the content
	// scan skipping an empty attr before finding a real one.
	buf := &bytes.Buffer{}
	h := newLeaf(buf).WithGroup("g1").WithGroup("g2")
	require.NoError(t, h.Handle(context.Background(), makeRecord(logutil.LevelInfo, "m", slog.Attr{}, slog.String("k", "v"))))

	m := decodeJSON(t, buf.Bytes())
	g1, _ := m["g1"].(map[string]any)
	g2, _ := g1["g2"].(map[string]any)
	require.Equal(t, "v", g2["k"])

	// WithAttrs under an open group nests below it.
	buf.Reset()
	h = newLeaf(buf).WithGroup("g").WithAttrs([]slog.Attr{slog.String("a", "1")})
	require.NoError(t, h.Handle(context.Background(), makeRecord(logutil.LevelInfo, "m", slog.Int("b", 2))))

	m = decodeJSON(t, buf.Bytes())
	g, _ := m["g"].(map[string]any)
	require.Equal(t, "1", g["a"])
	require.InDelta(t, 2, g["b"], 1e-9)

	// An empty innermost subgroup is skipped, but the non-empty outer group is kept.
	buf.Reset()
	h = newLeaf(buf).WithGroup("g1").WithAttrs([]slog.Attr{slog.String("x", "1")}).WithGroup("g2")
	require.NoError(t, h.Handle(context.Background(), makeRecord(logutil.LevelInfo, "m")))

	m = decodeJSON(t, buf.Bytes())
	g1, _ = m["g1"].(map[string]any)
	require.Equal(t, "1", g1["x"])
	require.NotContains(t, g1, "g2", "an empty subgroup must not appear as an empty object")
}

func Test_zerologHandler_emptyGroupOmitted(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	h := newLeaf(buf).WithGroup("g")
	require.NoError(t, h.Handle(context.Background(), makeRecord(logutil.LevelInfo, "m")))

	m := decodeJSON(t, buf.Bytes())
	require.NotContains(t, m, "g", "a group with no fields must be omitted entirely")
}

func Test_zerologHandler_emptyAttrsElided(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	h := newLeaf(buf)

	rec := makeRecord(logutil.LevelInfo, "m",
		slog.Attr{},                          // zero attr
		grp("empty"),                         // empty group
		grp("nested", slog.Attr{}, grp("d")), // group whose members are all empty
		slog.String("keep", "yes"),
	)

	require.NoError(t, h.Handle(context.Background(), rec))

	m := decodeJSON(t, buf.Bytes())
	require.NotContains(t, m, "empty")
	require.NotContains(t, m, "nested")
	require.Equal(t, "yes", m["keep"])
}

func Test_zerologHandler_withAttrsRootBaking(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	h := newLeaf(buf).WithAttrs([]slog.Attr{
		slog.String("s", "x"),
		slog.Int64("i", -3),
		slog.Uint64("u", 7),
		slog.Float64("f", 1.5),
		slog.Bool("b", true),
		slog.Duration("d", time.Second),
		slog.Time("t", time.Unix(0, 0).UTC()),
		slog.Any("err", errors.New("e")),
		slog.Any("any", map[string]any{"z": 1}),
		grp("g", slog.String("k", "v")),
		grp("", slog.String("inlined", "yes")),
		{},
		grp("empty"),
	})

	require.NoError(t, h.Handle(context.Background(), makeRecord(logutil.LevelInfo, "m")))

	m := decodeJSON(t, buf.Bytes())
	require.Equal(t, "x", m["s"])
	require.InDelta(t, -3, m["i"], 1e-9)
	require.InDelta(t, 7, m["u"], 1e-9)
	require.InEpsilon(t, 1.5, m["f"], 1e-9)
	require.Equal(t, true, m["b"])
	require.Contains(t, m, "d")
	require.Equal(t, "1970-01-01T00:00:00Z", m["t"])
	require.Equal(t, "e", m["err"])
	require.Equal(t, map[string]any{"z": float64(1)}, m["any"])
	require.Equal(t, map[string]any{"k": "v"}, m["g"])
	require.Equal(t, "yes", m["inlined"])
	require.NotContains(t, m, "empty")
}

func Test_zerologHandler_noopDerivations(t *testing.T) {
	t.Parallel()

	h := newLeaf(io.Discard)

	require.Same(t, h, h.WithAttrs(nil), "WithAttrs with no attrs must return the receiver")
	require.Same(t, h, h.WithGroup(""), "WithGroup with an empty name must return the receiver")
}

func Test_zerologHandler_enabled(t *testing.T) {
	t.Parallel()

	h := &zerologHandler{minLevel: logutil.LevelInfo}

	require.True(t, h.Enabled(context.Background(), logutil.LevelInfo))
	require.True(t, h.Enabled(context.Background(), logutil.LevelError))
	require.False(t, h.Enabled(context.Background(), logutil.LevelDebug))
}

func Test_zerologHandler_source(t *testing.T) {
	t.Parallel()

	// A record with no caller PC must not emit a source field.
	buf := &bytes.Buffer{}
	h := &zerologHandler{logger: zerolog.New(buf).Level(zerolog.TraceLevel), minLevel: logutil.LevelTrace, source: true}
	require.NoError(t, h.Handle(context.Background(), slog.NewRecord(time.Time{}, logutil.LevelInfo, "m", 0)))
	require.NotContains(t, decodeJSON(t, buf.Bytes()), "source")

	// A record with a caller PC emits function/file/line.
	buf.Reset()

	pc, _, _, ok := runtime.Caller(0)
	require.True(t, ok)
	require.NoError(t, h.Handle(context.Background(), slog.NewRecord(time.Time{}, logutil.LevelInfo, "m", pc)))

	src, ok := decodeJSON(t, buf.Bytes())["source"].(map[string]any)
	require.True(t, ok)
	require.Contains(t, src, "function")
	require.Contains(t, src, "file")
	require.Contains(t, src, "line")
}

// Test_zerologHandler_levelNames verifies the emitted "level" field carries the full syslog
// severity name (matching logutil), with exactly one level field and no collapse onto zerolog's
// fixed set (e.g. Critical stays "critical", not "error").
func Test_zerologHandler_levelNames(t *testing.T) {
	t.Parallel()

	for _, lv := range []logutil.LogLevel{
		logutil.LevelEmergency, logutil.LevelAlert, logutil.LevelCritical, logutil.LevelError,
		logutil.LevelWarning, logutil.LevelNotice, logutil.LevelInfo, logutil.LevelDebug, logutil.LevelTrace,
	} {
		buf := &bytes.Buffer{}

		require.NotPanics(t, func() {
			require.NoError(t, newLeaf(buf).Handle(context.Background(), makeRecord(lv, "m")))
		}, "level %s must not terminate the process", logutil.LevelName(lv))

		require.Equal(t, 1, strings.Count(buf.String(), `"level"`), "exactly one level field")
		require.Equal(t, logutil.LevelName(lv), decodeJSON(t, buf.Bytes())["level"])
	}
}

// The following tests pin intentional behavior differences from the previous slog-zerolog
// implementation (documented in the package "Notes"), so any future change fails loudly.

func Test_zerologHandler_errorRendersAsString(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	rec := makeRecord(logutil.LevelError, "m",
		slog.Any("error", errors.New("boom")),
		slog.Any("err", errors.New("bang")),
	)
	require.NoError(t, newLeaf(buf).Handle(context.Background(), rec))

	m := decodeJSON(t, buf.Bytes())
	require.Equal(t, "boom", m["error"], "an error must render as its message string, not a structured object")
	require.Equal(t, "bang", m["err"])
}

func Test_zerologHandler_nilValueRendersNull(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	require.NoError(t, newLeaf(buf).Handle(context.Background(), makeRecord(logutil.LevelInfo, "m", slog.Any("x", nil))))

	m := decodeJSON(t, buf.Bytes())
	require.Contains(t, m, "x", "a nil value is emitted as an explicit null, not omitted")
	require.Nil(t, m["x"])
}

func Test_zerologHandler_fieldsKeepInsertionOrder(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	// Keys chosen so alphabetical sorting would reverse their emitted order.
	rec := makeRecord(logutil.LevelInfo, "m", slog.Int("zebra", 1), slog.Int("alpha", 2))
	require.NoError(t, newLeaf(buf).Handle(context.Background(), rec))

	out := buf.String()
	require.Less(t, strings.Index(out, `"zebra"`), strings.Index(out, `"alpha"`),
		"fields must keep insertion order, not be sorted alphabetically")
}

// Test_zerologHandler_zeroTimeOmitted pins slog's rule that a zero record time is not emitted,
// while a non-zero time is.
func Test_zerologHandler_zeroTimeOmitted(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	require.NoError(t, newLeaf(buf).Handle(context.Background(), slog.NewRecord(time.Time{}, logutil.LevelInfo, "m", 0)))
	require.NotContains(t, decodeJSON(t, buf.Bytes()), "time", "a zero record time must be omitted")

	buf.Reset()
	require.NoError(t, newLeaf(buf).Handle(context.Background(), slog.NewRecord(time.Unix(1, 0), logutil.LevelInfo, "m", 0)))
	require.Contains(t, decodeJSON(t, buf.Bytes()), "time", "a non-zero record time must be written")
}
