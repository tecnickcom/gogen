package logutil

import (
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// LogLevel is an alias for slog.Level to represent extended log severity levels.
type LogLevel = slog.Level

// Extended slog levels.
const (
	LevelEmergency LogLevel = 64 // (+) 0 - Emergency - System is unusable.
	LevelAlert     LogLevel = 32 // (+) 1 - Alert - Immediate action required.
	LevelCritical  LogLevel = 16 // (+) 2 - Critical - Critical conditions.
	LevelError     LogLevel = 8  // (=) 3 - Error - Error conditions.
	LevelWarning   LogLevel = 4  // (=) 4 - Warning - Warning conditions.
	LevelNotice    LogLevel = 2  // (+) 5 - Notice - Normal but noteworthy events.
	LevelInfo      LogLevel = 0  // (=) 6 - Informational - General informational messages.
	LevelDebug     LogLevel = -4 // (=) 7 - Debug - Detailed debugging information.
	LevelTrace     LogLevel = -8 // (+) Additional Trace level when supported.
)

// ParseLevel converts syslog-style level strings ("0"-"7", syslog names, or "trace") to log levels.
// For unrecognized input it returns LevelInfo together with an error, so a caller that
// ignores the error degrades to a safe, non-verbose level rather than to debug output.
func ParseLevel(l string) (LogLevel, error) {
	switch strings.ToLower(l) {
	// 0 - Emergency - System is unusable
	case "0", "emerg", "emergency":
		return LevelEmergency, nil
	// 1 - Alert - Immediate action required
	case "1", "alert":
		return LevelAlert, nil
	// 2 - Critical - Critical conditions
	case "2", "crit", "critical":
		return LevelCritical, nil
	// 3 - Error - Error conditions
	case "3", "err", "error":
		return LevelError, nil
	// 4 - Warning - Warning conditions
	case "4", "warn", "warning":
		return LevelWarning, nil
	// 5 - Notice - Normal but noteworthy events
	case "5", "notice":
		return LevelNotice, nil
	// 6 - Informational - General informational messages
	case "6", "info":
		return LevelInfo, nil
	// 7 - Debug - Detailed debugging information
	case "7", "debug":
		return LevelDebug, nil
	// Additional Trace level
	case "trace":
		return LevelTrace, nil
	}

	return LevelInfo, fmt.Errorf("invalid log level %q", l)
}

// ValidLevel reports whether the given log level is recognized.
func ValidLevel(l LogLevel) bool {
	switch l {
	case LevelEmergency, LevelAlert, LevelCritical, LevelError, LevelWarning, LevelNotice, LevelInfo, LevelDebug, LevelTrace:
		return true
	default:
		return false
	}
}

// replaceLevelName is a slog.HandlerOptions.ReplaceAttr callback that repairs the two built-in
// attributes slog would otherwise render badly. It leaves attributes inside groups untouched.
//
// The top-level "level" attribute is rendered with the syslog-style names from LevelName (e.g.
// "critical", "notice", "emergency") instead of slog's numeric-offset fallbacks ("ERROR+8", "INFO+2").
// Unrecognized level values are left untouched so slog's own "WARN+1"-style banding is preserved rather
// than reduced to a bare number.
//
// The top-level "time" attribute (the record's own timestamp) is rewritten as an RFC 3339 string when
// its year falls outside [0,9999], because slog's JSON encoder writes an "!ERROR:" string for such a
// value and then writes the value as well, putting two JSON strings under one key and making the line
// invalid (see slogSanitizeHandler, which repairs the same shape in an *attribute*; the record's
// timestamp never passes through it, since it is not an attribute). A record only carries such a time
// if it was hand-built (slog.Logger always stamps time.Now()), but a middleware, tee, or replay
// handler can hand one over.
func replaceLevelName(groups []string, a Attr) Attr {
	if len(groups) != 0 {
		return a
	}

	switch a.Key {
	case slog.LevelKey:
		if level, ok := a.Value.Any().(LogLevel); ok && ValidLevel(level) {
			a.Value = slog.StringValue(LevelName(level))
		}
	case slog.TimeKey:
		if t := a.Value; t.Kind() == slog.KindTime && !encodableJSONTime(t.Time()) {
			a.Value = slog.StringValue(t.Time().Format(time.RFC3339Nano))
		}
	}

	return a
}

// LevelName returns the string name of the specified log level (e.g., "error", "debug").
//
// slog levels are arbitrary integers, so a level that is not one of the named severities has no name
// here. It falls back to slog's own banded form ("INFO+1", "WARN-2"), never to a bare number: a bare
// number would collide with the numeric syslog vocabulary that ParseLevel accepts, where "1" means
// alert, so a record one notch above info would be read back as a near-fatal severity.
//
//nolint:cyclop
func LevelName(l LogLevel) string {
	switch l {
	case LevelEmergency:
		return "emergency"
	case LevelAlert:
		return "alert"
	case LevelCritical:
		return "critical"
	case LevelError:
		return "error"
	case LevelWarning:
		return "warning"
	case LevelNotice:
		return "notice"
	case LevelInfo:
		return "info"
	case LevelDebug:
		return "debug"
	case LevelTrace:
		return "trace"
	default:
		return l.String()
	}
}
