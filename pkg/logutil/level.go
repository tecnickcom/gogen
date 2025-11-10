package logutil

import (
	"fmt"
	"log/slog"
	"strings"
)

// Extended slog levels.
const (
	LevelEmergency slog.Level = 64 // (+) 0 - Emergency - System is unusable.
	LevelAlert     slog.Level = 32 // (+) 1 - Alert - Immediate action required.
	LevelCritical  slog.Level = 16 // (+) 2 - Critical - Critical conditions.
	LevelError     slog.Level = 8  // (=) 3 - Error - Error conditions.
	LevelWarning   slog.Level = 4  // (=) 4 - Warning - Warning conditions.
	LevelNotice    slog.Level = 2  // (+) 5 - Notice - Normal but noteworthy events.
	LevelInfo      slog.Level = 0  // (=) 6 - Informational - General informational messages.
	LevelDebug     slog.Level = -4 // (=) 7 - Debug - Detailed debugging information.
)

// LevelStrToSlog converts syslog standard level strings to log/slog levels.
// Syslog uses eight severity levels, ranging from 0 (Emergency) to 7 (Debug).
// The lower the number, the higher the priority.
func LevelStrToSlog(l string) (slog.Level, error) {
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
