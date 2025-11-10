package logutil

import (
	"fmt"
	"strings"
)

// LogFormat represents the logging output format.
type LogFormat int8

const (
	FormatNone    LogFormat = -1 // Discard the logs.
	FormatJSON    LogFormat = 0  // Prints the logs in JSON format.
	FormatConsole LogFormat = 1  // Prints the logs in a human friendly format.
)

// ParseFormat converts a string to a log format.
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

// ValidFormat returns true if the log format is valid.
func ValidFormat(f LogFormat) bool {
	switch f {
	case FormatNone, FormatJSON, FormatConsole:
		return true
	default:
		return false
	}
}
