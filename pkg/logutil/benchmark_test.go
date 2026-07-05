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
