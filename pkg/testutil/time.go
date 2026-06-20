package testutil

import (
	"regexp"
)

// Precompiled regular expressions for timestamp replacement (compiled once at package load).
var (
	regexDateTime      = regexp.MustCompile("([0-9]{4}\\-[0-9]{2}\\-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}[^\"]*)")
	regexUnixTimestamp = regexp.MustCompile("([0-9]{19})")
)

// ReplaceDateTime replaces RFC3339-like datetime substrings in src with repl.
// It is useful for deterministic assertions on JSON responses with dynamic timestamps.
func ReplaceDateTime(src, repl string) string {
	return regexDateTime.ReplaceAllString(src, repl)
}

// ReplaceUnixTimestamp replaces 19-digit Unix-nanosecond timestamps in src with repl.
// It is useful for deterministic assertions on JSON responses with dynamic timestamps.
func ReplaceUnixTimestamp(src, repl string) string {
	return regexUnixTimestamp.ReplaceAllString(src, repl)
}
