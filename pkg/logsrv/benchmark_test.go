package logsrv

import (
	"io"
	"log/slog"
	"testing"

	"github.com/tecnickcom/gogen/pkg/logutil"
)

// BenchmarkNewHandler_Ungrouped exercises the common, ungrouped zerolog-backed JSON path.
// It uses NewHandler (not NewLogger) to avoid mutating the global slog default.
func BenchmarkNewHandler_Ungrouped(b *testing.B) {
	cfg, err := logutil.NewConfig(logutil.WithOutWriter(io.Discard), logutil.WithFormat(logutil.FormatJSON), logutil.WithLevel(logutil.LevelInfo))
	if err != nil {
		b.Fatal(err)
	}

	l := slog.New(NewHandler(cfg))

	b.ReportAllocs()

	for b.Loop() {
		l.Info("message", "key1", "value1", "key2", 42, "key3", true)
	}
}

// BenchmarkNewHandler_Grouped exercises a logger derived with WithGroup. logsrv keeps the
// trace ID at the root via its converter, so grouping does not trigger a per-record rebuild.
func BenchmarkNewHandler_Grouped(b *testing.B) {
	cfg, err := logutil.NewConfig(logutil.WithOutWriter(io.Discard), logutil.WithFormat(logutil.FormatJSON), logutil.WithLevel(logutil.LevelInfo))
	if err != nil {
		b.Fatal(err)
	}

	l := slog.New(NewHandler(cfg)).WithGroup("g")

	b.ReportAllocs()

	for b.Loop() {
		l.Info("message", "key1", "value1", "key2", 42, "key3", true)
	}
}

// BenchmarkNewHandler_FormatNoneNoHook exercises the FormatNone discard short-circuit. With
// no hook it must stay ~single-digit ns/op (a DiscardHandler that reports Enabled == false).
func BenchmarkNewHandler_FormatNoneNoHook(b *testing.B) {
	cfg, err := logutil.NewConfig(logutil.WithFormat(logutil.FormatNone), logutil.WithLevel(logutil.LevelInfo))
	if err != nil {
		b.Fatal(err)
	}

	l := slog.New(NewHandler(cfg))

	b.ReportAllocs()

	for b.Loop() {
		l.Info("message", "key1", "value1", "key2", 42, "key3", true)
	}
}
