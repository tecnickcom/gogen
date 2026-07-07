package redact

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHTTPDataAuthorizationLineBranches(t *testing.T) {
	t.Parallel()

	// Non-authorization header is left untouched.
	require.Equal(t, []byte("X-Header: SECRET\n"), HTTPDataBytes([]byte("X-Header: SECRET\n")))

	// Authorization header value is redacted, trailing newline preserved.
	require.Equal(t, []byte("Authorization: ***\n"), HTTPDataBytes([]byte("Authorization: Bearer SECRET\n")))

	// Authorization header value is redacted without a trailing newline.
	require.Equal(t, []byte("Authorization: ***"), HTTPDataBytes([]byte("Authorization: Bearer SECRET")))
}

func TestSensitiveHeaderValueStart(t *testing.T) {
	t.Parallel()

	// No colon after the name: not a header line.
	_, ok := defaultRedactor.sensitiveHeaderValueStart([]byte("Authorization no-colon\n"), 0)
	require.False(t, ok)

	// Name runs to end of input without a colon.
	_, ok = defaultRedactor.sensitiveHeaderValueStart([]byte("Authorization"), 0)
	require.False(t, ok)

	// Empty name before the colon: not a header line.
	_, ok = defaultRedactor.sensitiveHeaderValueStart([]byte(": value\n"), 0)
	require.False(t, ok)

	_, ok = defaultRedactor.sensitiveHeaderValueStart([]byte(" : value\n"), 0)
	require.False(t, ok)

	// Names with non-header characters are left to the JSON/URL rules.
	_, ok = defaultRedactor.sensitiveHeaderValueStart([]byte(`{"Authorization":"SECRET"}`), 0)
	require.False(t, ok)

	// Header names without sensitive tokens are not matched.
	_, ok = defaultRedactor.sensitiveHeaderValueStart([]byte("X-Header: SECRET\n"), 0)
	require.False(t, ok)

	// Sensitive header names are matched; valueStart is just past ": ".
	valueStart, ok := defaultRedactor.sensitiveHeaderValueStart([]byte("Authorization: Bearer SECRET\n"), 0)
	require.True(t, ok)
	require.Equal(t, len("Authorization: "), valueStart)

	valueStart, ok = defaultRedactor.sensitiveHeaderValueStart([]byte("Proxy-Authorization: Basic SECRET\n"), 0)
	require.True(t, ok)
	require.Equal(t, len("Proxy-Authorization: "), valueStart)

	valueStart, ok = defaultRedactor.sensitiveHeaderValueStart([]byte("X-Api-Key: SECRET\n"), 0)
	require.True(t, ok)
	require.Equal(t, len("X-Api-Key: "), valueStart)

	// Optional whitespace before the colon is allowed.
	valueStart, ok = defaultRedactor.sensitiveHeaderValueStart([]byte("authorization : ApiKey=SECRET\n"), 0)
	require.True(t, ok)
	require.Equal(t, len("authorization : "), valueStart)
}

func TestHeaderValueEnd(t *testing.T) {
	t.Parallel()

	// LF line: value ends at the '\n'.
	src := []byte("X-Api-Key: SECRET\nnext")
	require.Equal(t, len("X-Api-Key: SECRET"), headerValueEnd(src, len("X-Api-Key: ")))

	// CRLF line: value ends before the '\r' so the terminator is preserved.
	src = []byte("X-Api-Key: SECRET\r\nnext")
	require.Equal(t, len("X-Api-Key: SECRET"), headerValueEnd(src, len("X-Api-Key: ")))

	// No terminator: value runs to end of input ('\r' at EOF is kept as value).
	src = []byte("X-Api-Key: SECRET")
	require.Equal(t, len(src), headerValueEnd(src, len("X-Api-Key: ")))

	// Empty value directly before the terminator.
	src = []byte("X-Api-Key:\n")
	require.Equal(t, len("X-Api-Key:"), headerValueEnd(src, len("X-Api-Key:")))
}

func TestHTTPDataProxyAuthorizationHeader(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  string
	}{
		{"Proxy-Authorization: Basic SECRET\n", "Proxy-Authorization: ***\n"},
		{"PROXY-AUTHORIZATION: Bearer SECRET", "PROXY-AUTHORIZATION: ***"},
	}

	for _, tc := range cases {
		require.Equal(t, expectedRedaction(tc.want), HTTPData(tc.input), "input: %s", tc.input)
	}
}

// TestHTTPDataAuthorizationCRLF verifies the redacted Authorization line keeps
// its original terminator instead of collapsing \r\n to \n.
func TestHTTPDataAuthorizationCRLF(t *testing.T) {
	t.Parallel()

	require.Equal(t,
		[]byte("Authorization: ***\r\nHost: x\r\n"),
		HTTPDataBytes([]byte("Authorization: Bearer SECRET\r\nHost: x\r\n")),
	)

	// LF-only and no-terminator variants remain correct.
	require.Equal(t, []byte("Authorization: ***\n"), HTTPDataBytes([]byte("Authorization: Bearer SECRET\n")))
	require.Equal(t, []byte("Authorization: ***"), HTTPDataBytes([]byte("Authorization: Bearer SECRET")))
}

func TestHTTPDataSensitiveHeaderNames(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  string
	}{
		// Sensitive header names beyond *authorization* redact their whole value.
		{"X-Api-Key: abc123SECRET\n", "X-Api-Key: ***\n"},
		{"X-Auth-Token: SECRET\n", "X-Auth-Token: ***\n"},
		{"X-Amz-Security-Token: SECRET\n", "X-Amz-Security-Token: ***\n"},
		{"X-Csrf-Token: SECRET\n", "X-Csrf-Token: ***\n"},
		{"Api-Key: SECRET\n", "Api-Key: ***\n"},
		{"Signature: SECRET\n", "Signature: ***\n"},
		{"password: SECRET\n", "password: ***\n"},
		// Cookie headers redact fully, even opaque values without '='.
		{"Cookie: SESSIONTOKEN\n", "Cookie: ***\n"},
		{"Cookie: sid=abc123; theme=dark\n", "Cookie: ***\n"},
		{"Set-Cookie: id=SECRET; Path=/; HttpOnly\n", "Set-Cookie: ***\n"},
		// CRLF terminators are preserved.
		{"X-Auth-Token: SECRET\r\nHost: x\r\n", "X-Auth-Token: ***\r\nHost: x\r\n"},
		// Non-sensitive header names stay visible.
		{"Host: example.com\n", "Host: example.com\n"},
		{"User-Agent: Go-http-client/1.1\n", "User-Agent: Go-http-client/1.1\n"},
		{"X-GOGEN-Trace-Id: abc123\n", "X-GOGEN-Trace-Id: abc123\n"},
		{"Content-Type: application/json\n", "Content-Type: application/json\n"},
	}

	for _, tc := range cases {
		require.Equal(t, expectedRedaction(tc.want), HTTPData(tc.input), "input: %s", tc.input)
	}
}
