/*
Package logsrv implements a zerolog-based logger compatible with the standard log/slog.

This is compatible with the configuration model at:
  - Nicola Asuni, 2014-08-11, "Software Logging Format",
    https://technick.net/guides/software/software_logging_format/

See also: github.com/tecnickcom/pkg/logutil
*/
package logsrv

import (
	"io"
	"log/slog"

	"github.com/rs/zerolog"
	szlog "github.com/samber/slog-zerolog/v2"
	"github.com/tecnickcom/gogen/pkg/logutil"
)

const (
	traceIDName = "trace_id"
)

// NewLogger configures a new slog logger with a zerolog backend.
func NewLogger(cfg *logutil.Config) *slog.Logger {
	w := writerByFormat(cfg.Format, cfg.Out)

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

	zl := zerolog.New(w).With().Timestamp().Str(traceIDName, cfg.TraceIDFn()).Logger()

	if cfg.HookFn != nil {
		hf := func(_ *zerolog.Event, level zerolog.Level, message string) {
			cfg.HookFn(SlogLevel(level), message)
		}
		zl = zl.Hook(zerolog.HookFunc(hf))
	}

	sh := szlog.Option{
		Level:  cfg.Level,
		Logger: &zl,
	}.NewZerologHandler().WithAttrs(cfg.CommonAttr)

	sl := slog.New(sh)

	slog.SetDefault(sl)

	return sl
}

// writerByFormat returns the io.Writer for the selected log format.
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
