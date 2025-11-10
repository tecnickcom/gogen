package logutil

import (
	"fmt"
	"log/slog"
	"strings"
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
)

// ParseLevel converts syslog standard level strings to log/slog levels.
// Syslog uses eight severity levels, ranging from 0 (Emergency) to 7 (Debug).
// The lower the number, the higher the priority.
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
	}

	return LevelDebug, fmt.Errorf("invalid log level %q", l)
}

// ValidLevel returns true if the log level is valid.
func ValidLevel(l LogLevel) bool {
	switch l {
	case LevelEmergency, LevelAlert, LevelCritical, LevelError, LevelWarning, LevelNotice, LevelInfo, LevelDebug:
		return true
	default:
		return false
	}
}
