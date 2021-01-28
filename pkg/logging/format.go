package logging

import (
	"fmt"
	"strings"
)

// Format represents the logging format to adopt.
type Format int8

const (
	noFormat Format = iota

	// ConsoleFormat will print the log in a human friendly format.
	ConsoleFormat

	// JSONFormat will print the log in a machine readable format.
	JSONFormat
)

// ParseFormat converts a string to a log format.
func ParseFormat(f string) (Format, error) {
	switch strings.ToLower(f) {
	case "console":
		return ConsoleFormat, nil
	case "json":
		return JSONFormat, nil
	}
	return noFormat, fmt.Errorf("invalid log format %q", f)
}
