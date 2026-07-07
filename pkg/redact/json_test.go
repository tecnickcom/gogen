package redact

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHTTPDataJSONNumericValues(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  string
	}{
		{`{"amount": 9999}`, `{"amount": "***"}`},
		{`{"cvv": true}`, `{"cvv": "***"}`},
		{`{"ssn": null}`, `{"ssn": "***"}`},
		{`{"balance": -1.5e3}`, `{"balance": "***"}`},
		{`{"password": 0}`, `{"password": "***"}`},
	}

	for _, tc := range cases {
		require.Equal(t, expectedRedaction(tc.want), HTTPData(tc.input), "input: %s", tc.input)
	}
}

func TestHTTPDataJSONEdgeBranches(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "unmatched quote",
			input: `prefix "password`,
			want:  `prefix "password`,
		},
		{
			name:  "quoted token without colon",
			input: `"password" x`,
			want:  `"password" x`,
		},
		{
			name:  "colon then only whitespace",
			input: `{"password":   `,
			want:  `{"password":   `,
		},
		{
			name:  "string value with escapes",
			input: `{"password":"va\\\"l"}`,
			want:  `{"password":"***"}`,
		},
		{
			name:  "false literal redaction",
			input: `{"cvv":false}`,
			want:  `{"cvv":"***"}`,
		},
		{
			name:  "empty array value redacted",
			input: `{"password":[]}`,
			want:  `{"password":"***"}`,
		},
		{
			name:  "unknown value type left visible",
			input: `{"password":@x}`,
			want:  `{"password":@x}`,
		},
		{
			name:  "unterminated container redacts to end of input",
			input: `{"password":{"a":"b"`,
			want:  `{"password":"***"`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedRedaction(tc.want), HTTPData(tc.input))
		})
	}
}

func TestHTTPDataJSONNonSensitivePreserved(t *testing.T) {
	t.Parallel()

	input := []byte(`{"reference":"VISIBLE"}`)
	want := []byte(`{"reference":"VISIBLE"}`)
	require.Equal(t, want, HTTPDataBytes(input))
}

func TestRedactionHelpersCoverageBranches(t *testing.T) {
	t.Parallel()

	t.Run("json value start no colon after key", func(t *testing.T) {
		t.Parallel()

		_, hasKV, done := findJSONValueStart([]byte(`"password" x`), len(`"password"`)-1)
		require.False(t, hasKV)
		require.False(t, done)
	})

	t.Run("json value start end of input", func(t *testing.T) {
		t.Parallel()

		_, hasKV, done := findJSONValueStart([]byte(`"password"`), len(`"password"`)-1)
		require.False(t, hasKV)
		require.True(t, done)
	})

	t.Run("json string parser with trailing backslash", func(t *testing.T) {
		t.Parallel()

		src := []byte{0x22, 'a', 0x5c, 0x22}

		// Unterminated escaped sequence should return end-of-input.
		require.Equal(t, len(src), parseJSONStringEnd(src, 0))
	})
}

func TestHTTPDataAuthorizationJSONKey(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  string
	}{
		{`{"Authorization":"Bearer SECRET"}`, `{"Authorization":"***"}`},
		{`{"authorization": "Basic SECRET"}`, `{"authorization": "***"}`},
		{`{"Proxy-Authorization": "SECRET"}`, `{"Proxy-Authorization": "***"}`},
		{`authorization=SECRET&reference=VISIBLE`, `authorization=***&reference=VISIBLE`},
	}

	for _, tc := range cases {
		require.Equal(t, expectedRedaction(tc.want), HTTPData(tc.input), "input: %s", tc.input)
	}
}

func TestLikelyJSONKeyStart(t *testing.T) {
	t.Parallel()

	require.True(t, defaultRedactor.likelyJSONKeyStart([]byte(`"k":1`), 0))
	require.True(t, defaultRedactor.likelyJSONKeyStart([]byte(`{"k":1}`), 1))
	require.True(t, defaultRedactor.likelyJSONKeyStart([]byte(`{"a":1, "b":2}`), 8))
	require.False(t, defaultRedactor.likelyJSONKeyStart([]byte(`{"a":"v"}`), 6))
}

func TestFindJSONStringClosingQuote(t *testing.T) {
	t.Parallel()

	q, ok := findJSONStringClosingQuote([]byte(`a\"b"x`), 0)
	require.True(t, ok)
	require.Equal(t, 4, q)

	_, ok = findJSONStringClosingQuote([]byte(`a\"b`), 0)
	require.False(t, ok)
}

func TestAppendRedactedSensitiveJSONAtNoClosingQuote(t *testing.T) {
	t.Parallel()

	_, _, ok := defaultRedactor.appendRedactedSensitiveJSONAt([]byte(`"password`), 0, nil)
	require.False(t, ok)

	_, _, ok = defaultRedactor.appendRedactedSensitiveJSONAt([]byte(`"password"`), 0, nil)
	require.False(t, ok)
}

// TestHTTPDataSensitiveContainerValues covers redaction of object/array values
// under a sensitive key, which previously leaked their nested contents.
func TestHTTPDataSensitiveContainerValues(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  string
	}{
		{`{"password":{"inner":"SECRET"}}`, `{"password":"***"}`},
		{`{"token":{"raw":"SECRET"}}`, `{"token":"***"}`},
		{`{"secret":["SECRET1","SECRET2"]}`, `{"secret":"***"}`},
		{`{"password":["SECRET"]}`, `{"password":"***"}`},
		// Nested objects collapse to a single marker.
		{`{"secret":{"x":{"y":"z"}}}`, `{"secret":"***"}`},
		// A closing delimiter inside a string must not end the span early.
		{`{"password":{"a":"}"}}`, `{"password":"***"}`},
		// Non-sensitive outer key: inner sensitive keys are still redacted individually.
		{`{"profile":{"password":"SECRET"}}`, `{"profile":{"password":"***"}}`},
		// Trailing content after the redacted container is preserved.
		{`{"secret":[1,2],"note":"VISIBLE"}`, `{"secret":"***","note":"VISIBLE"}`},
	}

	for _, tc := range cases {
		require.Equal(t, expectedRedaction(tc.want), HTTPData(tc.input), "input: %s", tc.input)
	}
}

func TestParseJSONContainerEnd(t *testing.T) {
	t.Parallel()

	// Balanced object.
	require.Equal(t, len(`{"a":"b"}`), parseJSONContainerEnd([]byte(`{"a":"b"}`), 0))

	// Balanced array with trailing bytes.
	require.Equal(t, len(`[1,2]`), parseJSONContainerEnd([]byte(`[1,2],x`), 0))

	// A delimiter inside a string does not close the container.
	require.Equal(t, len(`{"a":"}"}`), parseJSONContainerEnd([]byte(`{"a":"}"}`), 0))

	// Unbalanced input consumes to end of input (truncation-safe).
	require.Equal(t, len(`{"a":"b"`), parseJSONContainerEnd([]byte(`{"a":"b"`), 0))

	// Unterminated string inside the container behaves the same.
	require.Equal(t, len(`{"a":"b`), parseJSONContainerEnd([]byte(`{"a":"b`), 0))
}

// TestHTTPDataJSONMalformedValues verifies unquoted literals and numbers are
// only redacted when followed by a proper JSON delimiter, so malformed input
// is left intact instead of being spliced around the marker.
func TestHTTPDataJSONMalformedValues(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  string
	}{
		{`{"cvv":truex}`, `{"cvv":truex}`},
		{`{"cvv":falsey}`, `{"cvv":falsey}`},
		{`{"ssn":nullx}`, `{"ssn":nullx}`},
		{`{"amount":12abc}`, `{"amount":12abc}`},
		// Well-formed values keep redacting, incl. at end of input.
		{`{"cvv":true}`, `{"cvv":"***"}`},
		{`{"cvv":true , "x":1}`, `{"cvv":"***" , "x":1}`},
		{`{"amount":12`, `{"amount":"***"`},
	}

	for _, tc := range cases {
		require.Equal(t, expectedRedaction(tc.want), HTTPData(tc.input), "input: %s", tc.input)
	}
}

// TestJSONKeyDoesNotSpanLines covers the fuzz-found idempotency bug: raw line
// breaks are illegal inside JSON strings, so a key scan must never pair quotes
// across lines — after other rules rewrite an intervening line, such phantom
// keys would re-match differently on a second pass.
func TestJSONKeyDoesNotSpanLines(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  string
	}{
		// The fuzz-found case: quotes re-pair across lines after the header
		// rule rewrites the middle line.
		{"\"\nAuth:\"\n\":0", "\"\nAuth:***\n\":0"},
		// A literal multi-line "key" is not treated as a key.
		{"\"pass\nword\": \"x\"", "\"pass\nword\": \"x\""},
		// Same-line keys keep redacting.
		{`{"password": "x"}`, `{"password": "***"}`},
	}

	for _, tc := range cases {
		require.Equal(t, expectedRedaction(tc.want), HTTPData(tc.input), "input: %s", tc.input)

		once := HTTPData(tc.input)
		require.Equal(t, once, HTTPData(once), "not idempotent for input: %s", tc.input)
	}
}

// TestJSONKeyAfterRedactedPrefix covers the fuzz-found idempotency class: a
// URL-encoded value redacted just before a JSON key replaces the key's
// structural context ('{', ',') with the marker, so the marker itself must
// count as key context or the second pass exposes the key's interior to the
// other rules (e.g. card digits).
func TestJSONKeyAfterRedactedPrefix(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  string
	}{
		{`sid={"pass 4000000000000 ":0`, `sid=***"pass 4000000000000 ":"***"`},
		{`sid=x {"pass 4000000000000":1`, `sid=***"pass 4000000000000":"***"`},
		// Literal marker text before a quoted sensitive pair is key context.
		{`***"password":"x"`, `***"password":"***"`},
		// One or two asterisks are not a marker.
		{`**"password":"x"`, `**"password":"x"`},
		// A marker-length tail that is not the marker is not key context.
		{`x**"password":"x"`, `x**"password":"x"`},
	}

	for _, tc := range cases {
		require.Equal(t, expectedRedaction(tc.want), HTTPData(tc.input), "input: %s", tc.input)

		once := HTTPData(tc.input)
		require.Equal(t, once, HTTPData(once), "not idempotent for input: %s", tc.input)
	}
}
