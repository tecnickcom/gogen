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
	"sync"

	"github.com/rs/zerolog"
	szlog "github.com/samber/slog-zerolog/v2"
	"github.com/tecnickcom/gogen/pkg/logutil"
)

const (
	traceIDName = "trace_id"
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

// NewLogger constructs a slog.Logger backed by zerolog, configured via logutil.Config.
// Applies format selection, attributes, trace-ID injection, hooks, and level mapping.
// Sets the returned logger as the process-wide slog default.
//
// The trace ID is resolved once, at construction time, via cfg.TraceIDFn and
// embedded as a fixed field on every record from the returned logger. This is
// intentional and matches the logutil model: callers that need a per-record
// trace ID should derive child loggers with logger.With instead. A nil
// TraceIDFn is valid (as in logutil) and simply omits the trace ID field.
//
// The hook (cfg.HookFn) is invoked at the slog layer, before the record is
// handed to zerolog, so it receives the original record level (e.g.
// logutil.LevelNotice or logutil.LevelCritical) rather than the collapsed
// zerolog level (Notice->Info, Critical->Error).
func NewLogger(cfg *logutil.Config) *slog.Logger {
	w := writerByFormat(cfg.Format, cfg.Out)

	setLogLevels()

	zctx := zerolog.New(w).With().Timestamp()

	// logutil treats a nil TraceIDFn as valid (no trace ID field); mirror
	// that here instead of panicking on a nil function call.
	if cfg.TraceIDFn != nil {
		zctx = zctx.Str(traceIDName, cfg.TraceIDFn())
	}

	zl := zctx.Logger()

	sh := szlog.Option{
		Level:  cfg.Level,
		Logger: &zl,
	}.NewZerologHandler().WithAttrs(cfg.CommonAttr)

	// Wrap the handler (as logutil does) instead of hooking the zerolog event:
	// a zerolog hook would only see the zerolog-collapsed level, losing the
	// Notice and Critical severities.
	if cfg.HookFn != nil {
		sh = logutil.NewSlogHookHandler(sh, cfg.HookFn)
	}

	sl := slog.New(sh)

	slog.SetDefault(sl)

	return sl
}

// writerByFormat returns the zerolog output writer for the specified format (JSON, console, or discard).
func writerByFormat(f logutil.LogFormat, w io.Writer) io.Writer {
	switch f {
	case logutil.FormatJSON:
		return w
	case logutil.FormatConsole:
		return zerolog.ConsoleWriter{Out: w}
	case logutil.FormatNone:
		return io.Discard
	default:
		return w
	}
}
