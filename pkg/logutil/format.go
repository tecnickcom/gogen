package logutil

import (
	"fmt"
	"strings"
)

// LogFormat selects how log records are encoded for output.
type LogFormat int8

const (
	FormatNone    LogFormat = -1 // Discard the logs.
	FormatJSON    LogFormat = 0  // Prints the logs in JSON format.
	FormatConsole LogFormat = 1  // Prints the logs in a human friendly format.
)

// ParseFormat converts a string ("json", "console", "none"/"discard"/"noop") to a log format.
// Returns error for unrecognized input.
func ParseFormat(f string) (LogFormat, error) {
	switch strings.ToLower(f) {
	case "json":
		return FormatJSON, nil
	case "console":
		return FormatConsole, nil
	case "none", "discard", "noop":
		return FormatNone, nil
	}

	return FormatNone, fmt.Errorf("invalid log format %q", f)
}

// ValidFormat reports whether the given log format is recognized.
func ValidFormat(f LogFormat) bool {
	switch f {
	case FormatNone, FormatJSON, FormatConsole:
		return true
	default:
		return false
	}
}
