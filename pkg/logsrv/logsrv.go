/*
Package logsrv provides a high-performance zerolog backend exposed through the
standard log/slog API.

# Problem

Teams often want the ecosystem compatibility of slog while still leveraging
zerolog's speed and compact structured output. Without an adapter layer,
applications end up with mixed logging APIs, inconsistent severity mapping, and
duplicated setup logic across services.

# Solution

This package bridges [log/slog] and zerolog using
github.com/samber/slog-zerolog/v2, while reusing the shared configuration model
from gogen's logutil package.

[NewLogger] creates a slog.Logger backed by zerolog and applies:
  - log format selection (JSON, console, discard),
  - common structured attributes,
  - trace ID injection,
  - optional hook execution,
  - and syslog-style level mapping from logutil levels to zerolog levels.

# Compatibility

The logging model is compatible with:
  - Nicola Asuni, 2014-08-11, "Software Logging Format",
    https://technick.net/guides/software/software_logging_format/

See also:
  - github.com/tecnickcom/gogen/pkg/logutil

# Benefits

logsrv lets applications keep the standard slog interface while using zerolog's
performance characteristics and structured logging ergonomics.
*/
package logsrv

import (
	"io"
	"log/slog"
	"os"
	"sync"

	"github.com/rs/zerolog"
	szlog "github.com/samber/slog-zerolog/v2"
	"github.com/tecnickcom/gogen/pkg/logutil"
)

// logLevelsOnce guards the single, process-wide initialization of the
// szlog.LogLevels map. The slog-zerolog handler reads that package global at
// log time (with no per-handler override hook), so it must be written exactly
// once to avoid a data race with loggers created by earlier NewLogger calls.
//
//nolint:gochecknoglobals // required to set szlog.LogLevels exactly once, race-free.
var logLevelsOnce sync.Once

// setLogLevels installs the syslog-style logutil-to-zerolog level mapping into
// the szlog.LogLevels process global exactly once.
//
// The mapping is constant, so a single write is sufficient and prevents the
// data race that arises from reassigning the global on every NewLogger call
// while previously created handlers concurrently read it at log time.
func setLogLevels() {
	logLevelsOnce.Do(func() {
		szlog.LogLevels = map[logutil.LogLevel]zerolog.Level{
			logutil.LevelEmergency: zerolog.PanicLevel,
			logutil.LevelAlert:     zerolog.FatalLevel,
			logutil.LevelCritical:  zerolog.ErrorLevel,
			logutil.LevelError:     zerolog.ErrorLevel,
			logutil.LevelWarning:   zerolog.WarnLevel,
			logutil.LevelNotice:    zerolog.InfoLevel,
			logutil.LevelInfo:      zerolog.InfoLevel,
			logutil.LevelDebug:     zerolog.DebugLevel,
			logutil.LevelTrace:     zerolog.TraceLevel,
		}
	})
}

// NewLogger constructs a slog.Logger backed by zerolog, configured via logutil.Config,
// and installs it as the process-wide slog default.
//
// Use [NewHandler] (for example slog.New(logsrv.NewHandler(cfg))) when a logger is
// needed without replacing the global default.
//
// A nil cfg falls back to logutil.DefaultConfig. See [NewHandler] for the details of
// format selection, attributes, trace-ID injection, hooks, and level mapping.
func NewLogger(cfg *logutil.Config) *slog.Logger {
	sl := slog.New(NewHandler(cfg))

	slog.SetDefault(sl)

	return sl
}

// NewHandler constructs the slog.Handler backing a logsrv logger, without mutating any
// global logger state. Applies format selection, common attributes, trace-ID injection,
// hooks, and the syslog-style logutil-to-zerolog level mapping. A nil cfg falls back to
// logutil.DefaultConfig, and a nil Out writer falls back to os.Stderr, so construction
// never yields a handler that panics on the first write.
//
// The trace ID is resolved per record via cfg.TraceIDFn (matching the logutil model), so
// a dynamic TraceIDFn reflects the current request/context on every line rather than being
// frozen at construction. It is written at the root of every record — even for loggers
// derived with WithGroup — via the slog-zerolog converter. A nil TraceIDFn is valid and
// simply omits the trace ID field.
//
// The hook (cfg.HookFn) is invoked at the slog layer, before the record is handed to
// zerolog, so it receives the original record level (e.g. logutil.LevelNotice or
// logutil.LevelCritical) rather than the collapsed zerolog level.
//
// Note: zerolog exposes a fixed set of levels, so the emitted "level" field collapses the
// extended severities — Critical and Error both read "error", Notice and Info both read
// "info", Emergency reads "panic", Alert reads "fatal". The hook and cfg.Level still carry
// the original severity; only the rendered zerolog level label is collapsed.
//
// Note: the level mapping is installed into the process-global szlog.LogLevels map (once,
// race-safe); this affects any other code in the process that uses slog-zerolog directly.
func NewHandler(cfg *logutil.Config) slog.Handler {
	if cfg == nil {
		cfg = logutil.DefaultConfig()
	}

	// FormatNone with no hook has nothing to write and no side effect to fire, so a
	// zero-cost DiscardHandler (Enabled == false) is used instead of running the full
	// zerolog encode path into io.Discard on every record.
	if cfg.Format == logutil.FormatNone && cfg.HookFn == nil {
		return slog.DiscardHandler
	}

	out := cfg.Out
	if out == nil {
		out = os.Stderr
	}

	setLogLevels()

	// No timestamp on the zerolog context: the slog-zerolog handler stamps every
	// record with the slog record time, so adding one here would duplicate the
	// "time" field. Common attributes are applied via the handler's WithAttrs.
	zl := zerolog.New(writerByFormat(cfg.Format, out))

	opt := szlog.Option{
		Level:     cfg.Level,
		Logger:    &zl,
		AddSource: cfg.Source,
	}

	// Inject the trace ID via the converter so it lands at the root of every record,
	// resolved per record, and stays at the root even for loggers derived with
	// WithGroup (a slog attribute would instead nest inside the open group). A nil
	// TraceIDFn leaves the default converter in place, omitting the field.
	if cfg.TraceIDFn != nil {
		opt.Converter = traceIDConverter(cfg.TraceIDFn)
	}

	h := opt.NewZerologHandler().WithAttrs(cfg.CommonAttr)

	// Wrap the handler (as logutil does) instead of hooking the zerolog event:
	// a zerolog hook would only see the zerolog-collapsed level, losing the
	// Notice and Critical severities.
	if cfg.HookFn != nil {
		h = logutil.NewSlogHookHandler(h, cfg.HookFn)
	}

	return h
}

// traceIDConverter returns a slog-zerolog converter that adds the trace ID resolved from
// fn to the root of every record's field map. Placing it in the converter output (rather
// than as a slog attribute) keeps it at the root even when the logger is derived with
// WithGroup.
//
// If the record already yields a root-level trace ID (a caller logged the reserved
// TraceIDKey), the caller's value is kept and no second one is injected, so the output
// never carries a duplicate trace_id — matching logutil.
func traceIDConverter(fn logutil.TraceIDFunc) szlog.Converter {
	return func(
		addSource bool,
		replaceAttr func(groups []string, a slog.Attr) slog.Attr,
		loggerAttr []slog.Attr,
		groups []string,
		record *slog.Record,
	) map[string]any {
		out := szlog.DefaultConverter(addSource, replaceAttr, loggerAttr, groups, record)
		if _, ok := out[logutil.TraceIDKey]; !ok {
			out[logutil.TraceIDKey] = fn()
		}

		return out
	}
}

// writerByFormat returns the zerolog output writer for the specified format (JSON, console, or discard).
func writerByFormat(f logutil.LogFormat, w io.Writer) io.Writer {
	switch f {
	case logutil.FormatJSON:
		return w
	case logutil.FormatConsole:
		// Colorize only when the destination is a terminal, so console output written
		// to a file or pipe does not embed raw ANSI escape sequences.
		return zerolog.ConsoleWriter{Out: w, NoColor: !isTerminalWriter(w)}
	case logutil.FormatNone:
		return io.Discard
	default:
		return w
	}
}

// isTerminalWriter reports whether w is a terminal (character device). Non-terminal
// writers (files, pipes, in-memory buffers) return false so console output is emitted
// without color escapes.
//
// It only recognizes a bare *os.File: a terminal wrapped in a decorator (e.g. a
// bufio.Writer) is treated as non-terminal and rendered without color. This is a
// deliberate, dependency-free heuristic (golang.org/x/term is not an allowed import);
// callers needing precise control should pass the terminal *os.File directly.
func isTerminalWriter(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}

	info, err := f.Stat()

	return err == nil && info.Mode()&os.ModeCharDevice != 0
}
