package logutil

import (
	"io"
	"log/slog"
	"testing"
)

// BenchmarkSlogLogger_Ungrouped exercises the common, ungrouped JSON path. It must stay
// allocation-free (0 allocs/op) — that is the fast-path guarantee.
func BenchmarkSlogLogger_Ungrouped(b *testing.B) {
	cfg, err := NewConfig(WithOutWriter(io.Discard), WithFormat(FormatJSON), WithLevel(LevelInfo))
	if err != nil {
		b.Fatal(err)
	}

	l := cfg.SlogLogger()

	b.ReportAllocs()

	for b.Loop() {
		l.Info("message", "key1", "value1", "key2", 42, "key3", true)
	}
}

// BenchmarkSlogLogger_UngroupedWideRecord exercises the ungrouped JSON path for a record carrying
// more attributes than slog.Record's five inline slots — the shape of an ordinary request log line,
// and the one the three-attribute benchmarks above cannot see.
//
// It guards the record rebuild the trace handler performs to place the trace ID at the root (see
// leadRecord): handing the record's attributes to AddAttrs one at a time instead of in a single call
// makes it regrow its overflow slice from scratch on each, which costs an allocation per attribute on
// exactly this shape while leaving every three-attribute benchmark at 0 allocs/op.
func BenchmarkSlogLogger_UngroupedWideRecord(b *testing.B) {
	cfg, err := NewConfig(WithOutWriter(io.Discard), WithFormat(FormatJSON), WithLevel(LevelInfo))
	if err != nil {
		b.Fatal(err)
	}

	l := cfg.SlogLogger()

	b.ReportAllocs()

	for b.Loop() {
		l.Info("message",
			"method", "GET", "path", "/v1/items", "status", 200, "bytes", 4096,
			"duration_ms", 12, "remote", "10.0.0.1", "user", "u-1", "shard", 3)
	}
}

// BenchmarkSlogLogger_Grouped exercises the group-aware slow path, which rebuilds the
// downstream handler chain per record to keep the trace ID at the root.
func BenchmarkSlogLogger_Grouped(b *testing.B) {
	cfg, err := NewConfig(WithOutWriter(io.Discard), WithFormat(FormatJSON), WithLevel(LevelInfo))
	if err != nil {
		b.Fatal(err)
	}

	l := cfg.SlogLogger().WithGroup("g")

	b.ReportAllocs()

	for b.Loop() {
		l.Info("message", "key1", "value1", "key2", 42, "key3", true)
	}
}

// BenchmarkSlogLogger_GroupedCommonAttr exercises the group-aware slow path for a logger that also
// carries common attributes — the shape every service in this repo builds. It guards the rule that
// CommonAttr is preformatted into the handler once (see Config.SlogHandler): applying it through the
// trace wrapper instead would make the per-record replay re-encode it on every line, which costs
// several allocations per record and is invisible to the ungrouped and no-CommonAttr benchmarks.
func BenchmarkSlogLogger_GroupedCommonAttr(b *testing.B) {
	cfg, err := NewConfig(
		WithOutWriter(io.Discard),
		WithFormat(FormatJSON),
		WithLevel(LevelInfo),
		WithCommonAttr(slog.String("service", "svc"), slog.String("env", "prod"), slog.String("version", "1")),
	)
	if err != nil {
		b.Fatal(err)
	}

	l := cfg.SlogLogger().WithGroup("g")

	b.ReportAllocs()

	for b.Loop() {
		l.Info("message", "key1", "value1", "key2", 42, "key3", true)
	}
}

// BenchmarkSlogHandler_FormatNoneNoHook exercises the FormatNone discard short-circuit. With
// no hook it must stay ~single-digit ns/op (a DiscardHandler that reports Enabled == false).
func BenchmarkSlogHandler_FormatNoneNoHook(b *testing.B) {
	cfg, err := NewConfig(WithFormat(FormatNone), WithLevel(LevelInfo))
	if err != nil {
		b.Fatal(err)
	}

	l := cfg.SlogLogger()

	b.ReportAllocs()

	for b.Loop() {
		l.Info("message", "key1", "value1", "key2", 42, "key3", true)
	}
}

// sink is a no-op io.Writer used so the bridge benchmark still exercises real JSON
// formatting (unlike io.Discard, which would invite a DiscardHandler that formats nothing).
type sink struct{}

func (sink) Write(p []byte) (int, error) { return len(p), nil }

// BenchmarkSlogWriter_Bridge exercises the standard log.Logger -> slog bridge path.
func BenchmarkSlogWriter_Bridge(b *testing.B) {
	l := slog.New(slog.NewJSONHandler(sink{}, &slog.HandlerOptions{Level: LevelInfo}))
	w := NewSlogWriter(l)
	line := []byte("bridged log line\n")

	b.ReportAllocs()

	for b.Loop() {
		_, _ = w.Write(line)
	}
}

// benchValuer is an ordinary LogValuer: it resolves to a plain value, so it exercises the resolve-and-
// substitute path without also exercising elision.
type benchValuer struct{ id string }

func (v benchValuer) LogValue() slog.Value { return slog.StringValue(v.id) }

// BenchmarkSlogLogger_RecordWithGroup exercises a record carrying a GROUP attribute — the shape the
// sanitizing handler actually has to walk, resolve and (when a subgroup renders nothing) rebuild.
//
// Every other benchmark in this file logs plain attributes, which take the kind-check fast path and
// never reach the filter's body, so none of them can see a change in its cost. Note _Grouped, below,
// derives the logger with WithGroup: its *record* attributes are still plain.
func BenchmarkSlogLogger_RecordWithGroup(b *testing.B) {
	cfg, err := NewConfig(WithOutWriter(io.Discard), WithFormat(FormatJSON), WithLevel(LevelInfo))
	if err != nil {
		b.Fatal(err)
	}

	l := cfg.SlogLogger()

	b.ReportAllocs()

	for b.Loop() {
		l.Info("message", slog.Group("g", slog.String("key1", "value1"), slog.Int("key2", 42)), slog.Bool("key3", true))
	}
}

// BenchmarkSlogLogger_RecordWithLogValuer exercises a record carrying a LogValuer, which the filter must
// resolve and substitute — the other shape that reaches its body.
func BenchmarkSlogLogger_RecordWithLogValuer(b *testing.B) {
	cfg, err := NewConfig(WithOutWriter(io.Discard), WithFormat(FormatJSON), WithLevel(LevelInfo))
	if err != nil {
		b.Fatal(err)
	}

	l := cfg.SlogLogger()

	b.ReportAllocs()

	for b.Loop() {
		l.Info("message", slog.Any("user", benchValuer{id: "u-1"}), slog.Int("key2", 42), slog.Bool("key3", true))
	}
}

// BenchmarkSlogLogger_Console exercises the text/console path, which the filter is equally load-bearing
// for — there an elided group renames the following field rather than breaking the syntax — and which no
// other benchmark covers.
func BenchmarkSlogLogger_Console(b *testing.B) {
	cfg, err := NewConfig(WithOutWriter(io.Discard), WithFormat(FormatConsole), WithLevel(LevelInfo))
	if err != nil {
		b.Fatal(err)
	}

	l := cfg.SlogLogger()

	b.ReportAllocs()

	for b.Loop() {
		l.Info("message", "key1", "value1", "key2", 42, "key3", true)
	}
}
