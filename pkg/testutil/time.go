package testutil

import (
	"regexp"
)

// Precompiled regular expressions for timestamp replacement (compiled once at package load).
var (
	// regexDateTime matches an RFC3339-like datetime: a date, the "T" separator, an
	// "HH:MM:SS" time, and optional fractional seconds and "Z"/"±hh:mm" timezone offset.
	// The tail is bounded to that grammar so the match never crosses newlines and never
	// depends on a following quote (unlike a broad "everything up to the next quote" match).
	regexDateTime = regexp.MustCompile(`[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(?:\.[0-9]+)?(?:Z|[+-][0-9]{2}:[0-9]{2})?`)

	// regexUnixTimestamp matches a standalone 19-digit integer (bounded by word
	// boundaries), so it never corrupts a longer number by replacing a 19-digit prefix.
	regexUnixTimestamp = regexp.MustCompile(`\b[0-9]{19}\b`)
)

// ReplaceDateTime replaces RFC3339-like datetime substrings in src with repl.
// It is useful for deterministic assertions on JSON responses with dynamic timestamps.
//
// The match is bounded to the RFC3339 grammar (date, "T", time, optional fractional
// seconds and timezone offset) and never spans newlines. Space-separated datetimes
// (for example "2006-01-02 15:04:05") are intentionally not matched.
func ReplaceDateTime(src, repl string) string {
	return regexDateTime.ReplaceAllString(src, repl)
}

// ReplaceUnixTimestamp replaces standalone 19-digit integers in src with repl.
// It is useful for deterministic assertions on JSON responses with dynamic
// Unix-nanosecond timestamps.
//
// This is a heuristic keyed only on digit count: any standalone 19-digit integer is
// replaced, including unrelated identifiers that happen to be exactly 19 digits long.
// Numbers with more or fewer than 19 digits are left untouched.
func ReplaceUnixTimestamp(src, repl string) string {
	return regexUnixTimestamp.ReplaceAllString(src, repl)
}
