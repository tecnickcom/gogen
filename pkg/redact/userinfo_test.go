package redact

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRedactURLUserinfoPassword(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  string
	}{
		// DSN credentials in prose/error messages: password redacted, scheme,
		// user and host preserved.
		{
			"dial error: postgres://app:hunter2@10.0.0.5/db?sslmode=disable",
			"dial error: postgres://app:***@10.0.0.5/db?sslmode=disable",
		},
		{"redis://user:PASS@cache:6379", "redis://user:***@cache:6379"},
		{"mongodb+srv://root:pw@cluster0.x.net", "mongodb+srv://root:***@cluster0.x.net"},
		// Password containing sub-delims is fully redacted.
		{"amqp://guest:p&s=w@mq:5672", "amqp://guest:***@mq:5672"},
		// Empty user or empty password still redacts the password span.
		{"tcp://:secret@h", "tcp://:***@h"},
		// Userinfo without a password is left untouched.
		{"ftp://anonymous@host", "ftp://anonymous@host"},
		// URLs without userinfo are untouched (port colons, paths).
		{"see https://example.com/path and more", "see https://example.com/path and more"},
		{"tcp://host:5432/db", "tcp://host:5432/db"},
		// '@' after a URL boundary does not bind to the scheme.
		{"http://host/x a@b", "http://host/x a@b"},
		// Inside a JSON string value under a non-sensitive key.
		{`{"url":"postgres://u:p@h/db"}`, `{"url":"postgres://u:***@h/db"}`},
		// Already-redacted output is stable (idempotency).
		{"postgres://app:***@h/db", "postgres://app:***@h/db"},
		// "://" at end of input.
		{"scheme://", "scheme://"},
	}

	for _, tc := range cases {
		require.Equal(t, expectedRedaction(tc.want), Default().String(tc.input), "input: %s", tc.input)
	}
}

func TestRedactURLUserinfoScanCap(t *testing.T) {
	t.Parallel()

	// An '@' beyond the bounded userinfo scan is not treated as credentials.
	long := "http://" + strings.Repeat("a", maxUserinfoScan) + ":p@host"
	require.Equal(t, long, Default().String(long))
}

func TestRedactUserinfoColonAfterDigits(t *testing.T) {
	t.Parallel()

	// A ':' reached directly after a digit run (times, ports, ratios) is not
	// a scheme separator and must be left untouched.
	require.Equal(t, "12:30:45", Default().String("12:30:45"))
	require.Equal(t, "ratio 1:2", Default().String("ratio 1:2"))
}

// TestRedactUserinfoLastAt verifies url.Parse semantics: the LAST '@'
// before a boundary delimits the userinfo, so passwords containing an
// unencoded '@' are hidden in full.
func TestRedactUserinfoLastAt(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  string
	}{
		{"http://user:p@ss@host/x", "http://user:***@host/x"},
		{"http://u:p@evil@h", "http://u:***@h"},
		// User-only userinfo followed by a port colon: no password to redact.
		{"http://u@h:80/x", "http://u@h:80/x"},
	}

	for _, tc := range cases {
		require.Equal(t, expectedRedaction(tc.want), Default().String(tc.input), "input: %s", tc.input)
	}
}

// TestUserinfoScanCapIsHardNoMatch verifies that hitting the scan cap before
// a natural boundary never matches: window-relative matching would make
// redaction depend on how much a previous pass shrank the preceding text
// (fuzz-found convergence bug with '@'s spaced beyond the cap).
func TestUserinfoScanCapIsHardNoMatch(t *testing.T) {
	t.Parallel()

	input := "://" + strings.Repeat("0", 25) + ":" + strings.Repeat("0", 260) +
		"@" + strings.Repeat("0", 230) + "@" + strings.Repeat("0", 25) + "@"

	require.Equal(t, input, Default().String(input))

	once := Default().String(input)
	require.Equal(t, once, Default().String(once))
}
