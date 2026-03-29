package testutil

import (
	"regexp"
)

// ReplaceDateTime replaces RFC3339-like datetime substrings in src with repl.
// It is useful for deterministic assertions on JSON responses with dynamic timestamps.
func ReplaceDateTime(src, repl string) string {
	re := regexp.MustCompile("([0-9]{4}\\-[0-9]{2}\\-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}[^\"]*)")
	return re.ReplaceAllString(src, repl)
}

// ReplaceUnixTimestamp replaces 19-digit Unix-nanosecond timestamps in src with repl.
// It is useful for deterministic assertions on JSON responses with dynamic timestamps.
func ReplaceUnixTimestamp(src, repl string) string {
	re := regexp.MustCompile("([0-9]{19})")
	return re.ReplaceAllString(src, repl)
}
