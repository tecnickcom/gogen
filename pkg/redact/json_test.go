package redact

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRedactJSONNumericValues(t *testing.T) {
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
		require.Equal(t, expectedRedaction(tc.want), Default().String(tc.input), "input: %s", tc.input)
	}
}

func TestRedactJSONEdgeBranches(t *testing.T) {
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
			require.Equal(t, expectedRedaction(tc.want), Default().String(tc.input))
		})
	}
}

func TestRedactJSONNonSensitivePreserved(t *testing.T) {
	t.Parallel()

	input := []byte(`{"reference":"VISIBLE"}`)
	want := []byte(`{"reference":"VISIBLE"}`)
	require.Equal(t, want, Default().Bytes(input))
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

func TestRedactAuthorizationJSONKey(t *testing.T) {
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
		require.Equal(t, expectedRedaction(tc.want), Default().String(tc.input), "input: %s", tc.input)
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

// TestRedactSensitiveContainerValues covers redaction of object/array values
// under a sensitive key, which previously leaked their nested contents.
func TestRedactSensitiveContainerValues(t *testing.T) {
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
		require.Equal(t, expectedRedaction(tc.want), Default().String(tc.input), "input: %s", tc.input)
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

// TestRedactJSONMalformedValues verifies unquoted literals and numbers are
// only redacted when followed by a proper JSON delimiter, so malformed input
// is left intact instead of being spliced around the marker.
func TestRedactJSONMalformedValues(t *testing.T) {
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
		require.Equal(t, expectedRedaction(tc.want), Default().String(tc.input), "input: %s", tc.input)
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
		require.Equal(t, expectedRedaction(tc.want), Default().String(tc.input), "input: %s", tc.input)

		once := Default().String(tc.input)
		require.Equal(t, once, Default().String(once), "not idempotent for input: %s", tc.input)
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
		require.Equal(t, expectedRedaction(tc.want), Default().String(tc.input), "input: %s", tc.input)

		once := Default().String(tc.input)
		require.Equal(t, once, Default().String(once), "not idempotent for input: %s", tc.input)
	}
}

// TestLabeledSecretJSON covers the {"name":"<secret-name>","value":<v>} shape
// (Kubernetes env, Docker inspect, AWS tags): the sibling value of a label key
// that names a secret is redacted, while the label itself stays readable unless
// it is a sensitive key in its own right.
func TestLabeledSecretJSON(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  string
	}{
		{`{"env":[{"name":"DB_PASSWORD","value":"hunter2"}]}`, `{"env":[{"name":"DB_PASSWORD","value":"***"}]}`},
		{`{"name":"DB_PASSWORD","value":"hunter2"}`, `{"name":"DB_PASSWORD","value":"***"}`},
		{`{"key":"api_key","value":"AKIAIOSFODNN7SECRET"}`, `{"key":"***","value":"***"}`},
		{`[{"Key":"apikey","Value":"SECRET"}]`, `[{"Key":"***","Value":"***"}]`}, // AWS-tag casing
		{`{"id":"password","value":123}`, `{"id":"password","value":"***"}`},     // non-string value
		{`{"name":"secret","val":"x"}`, `{"name":"secret","val":"***"}`},         // "val" alias
		// A non-secret label leaves the pair untouched.
		{`{"name":"LOG_LEVEL","value":"debug"}`, `{"name":"LOG_LEVEL","value":"debug"}`},
		{`{"name":"foo","value":"bar"}`, `{"name":"foo","value":"bar"}`},
		// No sibling value member: nothing to redact beyond the normal key rule.
		{`{"name":"DB_PASSWORD"}`, `{"name":"DB_PASSWORD"}`},
		{`{"key":"secretval"}`, `{"key":"***"}`}, // "key" is itself sensitive
	}

	for _, tc := range cases {
		require.Equal(t, expectedRedaction(tc.want), Default().String(tc.input), "input: %s", tc.input)

		once := Default().String(tc.input)
		require.Equal(t, once, Default().String(once), "not idempotent for input: %s", tc.input)
	}
}

// TestEscapedJSON covers a JSON document embedded as a string value inside
// another JSON document (captured request bodies / webhook payloads). A
// sensitive escaped key's escaped value is redacted; any inner escaping the
// conservative parser cannot handle is left untouched rather than mis-parsed.
func TestEscapedJSON(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  string
	}{
		{`{"body":"{\"password\":\"hunter2\"}"}`, `{"body":"{\"password\":\"***\"}"}`},
		{`{"req":"{\"user\":\"bob\",\"api_key\":\"AKIA\",\"token\":\"s3cr3t\"}"}`, `{"req":"{\"user\":\"bob\",\"api_key\":\"***\",\"token\":\"***\"}"}`},
		{`{"body":"{\"note\":\"visible\",\"secret\":\"hidden\"}"}`, `{"body":"{\"note\":\"visible\",\"secret\":\"***\"}"}`},
		// Non-sensitive escaped keys stay visible.
		{`{"body":"{\"note\":\"visible\"}"}`, `{"body":"{\"note\":\"visible\"}"}`},
		// A value with inner escaping is beyond the conservative parser: bail,
		// leaving the input unchanged (no mis-parse, no crash).
		{`{"body":"{\"password\":\"a\\\"b\"}"}`, `{"body":"{\"password\":\"a\\\"b\"}"}`},
		// Backslash-heavy content (a Windows path) must not be disturbed.
		{`{"path":"C:\\Users\\bob\\secret.txt"}`, `{"path":"C:\\Users\\bob\\secret.txt"}`},
	}

	for _, tc := range cases {
		require.Equal(t, tc.want, Default().String(tc.input), "input: %s", tc.input)

		// Convergence: two passes reach a fixed point.
		twice := Default().String(Default().String(tc.input))
		require.Equal(t, twice, Default().String(twice), "not convergent for input: %s", tc.input)
	}
}

// TestEscapedJSONBailPaths exercises the conservative parser's early exits: it
// must leave malformed or too-complex escaped input unchanged, never crashing
// or half-redacting.
func TestEscapedJSONBailPaths(t *testing.T) {
	t.Parallel()

	// Each input reaches the escaped-JSON dispatch (a `{\"` or `,\"`) but fails
	// one of the parse steps, so the output equals the input.
	unchanged := []string{
		`{\"password`,            // unterminated key
		`{\"password\" x`,        // no ':' after key
		`{\"password\":123`,      // value is not an escaped string
		`{\"password\":\"secret`, // unterminated value
		`{\"`,                    // key opens at end of input
		`,\"token\":\"`,          // value opens at end of input
		`{\"note\":\"visible\"}`, // non-sensitive key, left alone
	}
	for _, in := range unchanged {
		require.Equal(t, in, Default().String(in), "should be unchanged: %q", in)
	}
}

// TestLabeledSecretJSONBailPaths exercises the label/value parser's early exits.
func TestLabeledSecretJSONBailPaths(t *testing.T) {
	t.Parallel()

	unchanged := []string{
		`{"name":123,"value":"x"}`,         // label value is not a string
		`{"name":"password","other":"x"}`,  // sibling is not a value key
		`{"name":"password"}`,              // no sibling member
		`{"name":"password","value"}`,      // sibling has no value
		`{"label":"password","value":"x"}`, // not a recognized label key
		`{"name":"password`,                // label value string is never closed
		`{"name":"`,                        // label value is empty and unterminated
		`{"name":"password", 123}`,         // sibling member is not a quoted key
		`{"name":"password","value":}`,     // sibling value key has no parsable value
	}
	for _, in := range unchanged {
		require.Equal(t, in, Default().String(in), "should be unchanged: %q", in)
	}
}

// TestEscapedJSONNewlineInContentBails covers the line-break branch of the
// conservative escaped-string scanner: a raw newline cannot appear inside a
// JSON string, so the parser bails and leaves the input unchanged.
func TestEscapedJSONNewlineInContentBails(t *testing.T) {
	t.Parallel()

	in := "{\\\"pass\nword\\\":\\\"x\\\"}"
	require.Equal(t, in, Default().String(in))
}
