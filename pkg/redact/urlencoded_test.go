package redact

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHTTPDataURLEncodedNoFalsePositive(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  string
	}{
		// keyword in URL path must not cause a later param to be redacted
		{"GET /api/payment/receipt?reference=VISIBLE", "GET /api/payment/receipt?reference=VISIBLE"},
		// keyword in query param key should still be redacted
		{"GET /api/v1/status?session_id=SECRET", "GET /api/v1/status?session_id=***"},
		// keyword in path segment but innocent query param remains visible
		{"/authenticate?next=VISIBLE", "/authenticate?next=VISIBLE"},
	}

	for _, tc := range cases {
		require.Equal(t, expectedRedaction(tc.want), HTTPData(tc.input), "input: %s", tc.input)
	}
}

// TestHTTPDataURLFragmentParams verifies parameters in a URL fragment
// (OAuth 2.0 implicit-flow / SAML tokens) redact just like query params, with
// the path before the '#' left intact.
func TestHTTPDataURLFragmentParams(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  string
	}{
		{"https://app.example.com/cb#access_token=ya29.SECRET&token_type=Bearer", "https://app.example.com/cb#access_token=***&token_type=***"},
		{"https://app.example.com/cb#id_token=SECRET.VALUE", "https://app.example.com/cb#id_token=***"},
		// A space is not a value boundary (documented safe over-redaction), so a
		// trailing " HTTP/1.1" on the last param is consumed into the marker.
		{"GET /oauth/callback#access_token=SECRET HTTP/1.1", "GET /oauth/callback#access_token=***"},
		// A non-sensitive fragment key stays visible, and the path is untouched.
		{"/dashboard#tab=billing", "/dashboard#tab=billing"},
	}

	for _, tc := range cases {
		require.Equal(t, expectedRedaction(tc.want), HTTPData(tc.input), "input: %s", tc.input)
		// The fix must not disturb convergence.
		once := HTTPData(tc.input)
		require.Equal(t, once, HTTPData(once), "not idempotent: %s", tc.input)
	}
}

func TestHTTPDataURLEncodedSlashInKey(t *testing.T) {
	t.Parallel()

	input := []byte("GET /x/password=SECRET&password=SECRET")
	want := []byte("GET /x/password=SECRET&password=***")
	require.Equal(t, want, HTTPDataBytes(input))
}

func TestHTTPDataURLEncodedNoEquals(t *testing.T) {
	t.Parallel()

	input := []byte("GET /health HTTP/1.1")
	want := []byte("GET /health HTTP/1.1")
	require.Equal(t, want, HTTPDataBytes(input))
}

func TestHTTPDataURLEncodedNonSensitivePreserved(t *testing.T) {
	t.Parallel()

	input := []byte("reference=VISIBLE&note=PUBLIC")
	want := []byte("reference=VISIBLE&note=PUBLIC")
	require.Equal(t, want, HTTPDataBytes(input))
}

func TestHTTPDataURLEncodedKeyBoundaries(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  string
	}{
		// A sensitive word earlier in a prose line must not redact an unrelated
		// pair: the key is bounded by whitespace and pair separators.
		{"contact email:test@x.com reference=VISIBLE", "contact email:test@x.com reference=VISIBLE"},
		{"the amount is due ref=VISIBLE", "the amount is due ref=VISIBLE"},
		// The token adjacent to '=' still redacts.
		{"see docs token=SECRET", "see docs token=***"},
		// Semicolon-separated pairs are bounded individually.
		{"a=1; sid=SECRET", "a=1; sid=***"},
		{"x=1, password=SECRET", "x=1, password=***"},
	}

	for _, tc := range cases {
		require.Equal(t, expectedRedaction(tc.want), HTTPData(tc.input), "input: %s", tc.input)
	}
}

func TestHTTPDataURLEncodedValueBoundaries(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  string
	}{
		// CRLF terminators are preserved on redacted URL-encoded values.
		{"password=SECRET\r\nnext=ok", "password=***\r\nnext=ok"},
		{"a=1&token=SECRET\r\n", "a=1&token=***\r\n"},
		// A key=value pair embedded in a JSON string must not eat the string's
		// closing quote or the rest of the document.
		{`{"note":"a=b&password=c","next":"VISIBLE"}`, `{"note":"a=b&password=***","next":"VISIBLE"}`},
		// A double-quoted value is consumed through its closing quote.
		{`X-Data: password="secret value" trailing`, `X-Data: password=*** trailing`},
		// An unterminated quoted value is redacted to the line end.
		{"password=\"secret value\nnext=ok", "password=***\nnext=ok"},
		{`password="secret value`, `password=***`},
	}

	for _, tc := range cases {
		require.Equal(t, expectedRedaction(tc.want), HTTPData(tc.input), "input: %s", tc.input)
	}
}

// TestQuotedValueGluedWordChars covers the fuzz-found idempotency bug: word
// characters glued to a quoted value's closing quote must be consumed with the
// value, otherwise the first pass emits "***<char>" and a second pass swallows
// the line differently.
func TestQuotedValueGluedWordChars(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  string
	}{
		{`pass=""0`, `pass=***`},
		{`pass="x"0`, `pass=***`},
		{`token="v"tail rest`, `token=*** rest`},
		// Non-word bytes after the closing quote are preserved.
		{`pass="x" tail`, `pass=*** tail`},
		{`pass="x","next":"VISIBLE"`, `pass=***,"next":"VISIBLE"`},
	}

	for _, tc := range cases {
		require.Equal(t, expectedRedaction(tc.want), HTTPData(tc.input), "input: %s", tc.input)

		// The fix exists for idempotency: verify it directly.
		once := HTTPData(tc.input)
		require.Equal(t, once, HTTPData(once), "not idempotent for input: %s", tc.input)
	}
}

// TestUnquotedValueStopsAtAngleBrackets verifies an unquoted XML attribute
// value does not consume the tag close and the rest of the document.
func TestUnquotedValueStopsAtAngleBrackets(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  string
	}{
		{"<user password=abc>text</user>", "<user password=***>text</user>"},
		{"password=abc<next>", "password=***<next>"},
	}

	for _, tc := range cases {
		require.Equal(t, expectedRedaction(tc.want), HTTPData(tc.input), "input: %s", tc.input)

		once := HTTPData(tc.input)
		require.Equal(t, once, HTTPData(once), "not idempotent for input: %s", tc.input)
	}
}

// TestTrailingEqualsAfterRedactedPair covers the fuzz-found idempotency bug:
// '=' is a key boundary, so a bare '=' following a redacted pair must not
// re-match the sensitive key through the marker on a second pass.
func TestTrailingEqualsAfterRedactedPair(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  string
	}{
		{`token=""=`, `token=***=`},
		// Pair separators are value boundaries: consuming them would change
		// the structural context of what follows between redaction passes.
		// A quoted key containing '=' is owned by the URL rule on every pass.
		{`sid=,"sid=":0`, `sid=***,"sid=***`},
		{`"pass=""":0`, `"pass=***":0`},
		{`sid=abc,ref=1`, `sid=***,ref=1`},
		{`sid=abc;theme=dark`, `sid=***;theme=dark`},
		{`password=x==`, `password=***`}, // base64-style padding consumed with the value
		{`a=b=c`, `a=b=c`},               // non-sensitive keys unaffected by the boundary
		{`key=v=w`, `key=***`},           // sensitive value consumes through inner '='
	}

	for _, tc := range cases {
		require.Equal(t, expectedRedaction(tc.want), HTTPData(tc.input), "input: %s", tc.input)

		once := HTTPData(tc.input)
		require.Equal(t, once, HTTPData(once), "not idempotent for input: %s", tc.input)
	}
}

// TestKeyBoundarySupersetOfValueBoundary enforces the invariant that every
// byte able to terminate an unquoted value also fences the following key
// scan: a redacted value is replaced by inert marker bytes, so asymmetric
// boundaries would make key extraction differ between redaction passes.
func TestKeyBoundarySupersetOfValueBoundary(t *testing.T) {
	t.Parallel()

	for c := range 256 {
		if isURLValueBoundary(byte(c)) {
			require.Truef(t, isURLKeyBoundary(byte(c)),
				"value boundary %q is not a key boundary", byte(c))
		}
	}
}

// TestValueBoundaryBytesFenceFollowingKeys covers the fuzz-found case where a
// redacted value swallowed a byte ('/') that the following key scan depended
// on: with symmetric boundaries both passes extract the same keys.
func TestValueBoundaryBytesFenceFollowingKeys(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  string
	}{
		{"pass=/<pass=", "pass=***<pass=***"},
		{`pass="x".pass=y`, `pass=***.pass=***`},
	}

	for _, tc := range cases {
		require.Equal(t, expectedRedaction(tc.want), HTTPData(tc.input), "input: %s", tc.input)

		once := HTTPData(tc.input)
		require.Equal(t, once, HTTPData(once), "not idempotent for input: %s", tc.input)
	}
}
