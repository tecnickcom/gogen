package logsrv

import (
	"io"
	"log/slog"
	"testing"

	"github.com/tecnickcom/gogen/pkg/logutil"
)

// BenchmarkNewHandler_Ungrouped exercises the common, ungrouped zerolog-backed JSON path.
// It uses NewHandler (not NewLogger) to avoid mutating the global slog default. Writing
// attributes directly onto a zerolog Event, it must stay allocation-free (0 allocs/op).
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

// BenchmarkNewHandler_CommonAttrs verifies that baking the common attributes into the
// zerolog context keeps the per-record path allocation-free even with attributes present.
func BenchmarkNewHandler_CommonAttrs(b *testing.B) {
	cfg, err := logutil.NewConfig(
		logutil.WithOutWriter(io.Discard),
		logutil.WithFormat(logutil.FormatJSON),
		logutil.WithLevel(logutil.LevelInfo),
		logutil.WithCommonAttr(slog.String("service", "svc"), slog.String("env", "prod")),
	)
	if err != nil {
		b.Fatal(err)
	}

	l := slog.New(NewHandler(cfg))

	b.ReportAllocs()

	for b.Loop() {
		l.Info("message", "key1", "value1", "key2", 42, "key3", true)
	}
}

// BenchmarkNewHandler_GroupedEmpty logs under an open group with no attributes. The group is
// elided and the transient dictionary is recycled, so it must stay allocation-free (0 allocs) —
// a guard against pooled-event churn on the empty-group path.
func BenchmarkNewHandler_GroupedEmpty(b *testing.B) {
	cfg, err := logutil.NewConfig(logutil.WithOutWriter(io.Discard), logutil.WithFormat(logutil.FormatJSON), logutil.WithLevel(logutil.LevelInfo))
	if err != nil {
		b.Fatal(err)
	}

	l := slog.New(NewHandler(cfg)).WithGroup("g")

	b.ReportAllocs()

	for b.Loop() {
		l.Info("message")
	}
}

// BenchmarkNewHandler_Grouped exercises a logger derived with WithGroup, which nests the
// record's attributes under the open group.
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
