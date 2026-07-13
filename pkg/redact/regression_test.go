package redact

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestXMLContentScanIsLinear guards against a quadratic blow-up: a payload packed
// with sensitive XML elements each opening an unterminated comment/CDATA must not
// rescan the tail per tag.
func TestXMLContentScanIsLinear(t *testing.T) {
	t.Parallel()

	// Sized to complete quickly when linear but time out if the scan were
	// quadratic; the generous window tolerates the -race detector's overhead.
	for _, input := range [][]byte{
		[]byte(strings.Repeat("<token><![CDATA[yyyyyyyy", 30000)), // ~700 KB, no ]]>
		[]byte(strings.Repeat("<token><!--yyyyyyyy", 30000)),      // ~570 KB, no -->
	} {
		done := make(chan struct{})

		go func() {
			_ = Bytes(input)

			close(done)
		}()

		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Fatalf("XML content scan did not finish in 3s for a %d-byte input (quadratic?)", len(input))
		}
	}
}

// TestXMLContentScanCorrectness verifies the bounded scan still redacts legit
// comment/CDATA content and, crucially, that a large PLAIN (non-CDATA) secret is
// not left visible by the window.
func TestXMLContentScanCorrectness(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  string
	}{
		{`<password>a<![CDATA[hunter2]]></password>`, `<password>***</password>`},
		{`<password><!--c-->hunter2</password>`, `<password>***</password>`},
		{`user <token> expired`, `user <token> expired`}, // prose stays untouched
	}
	for _, tc := range cases {
		require.Equal(t, expectedRedaction(tc.want), HTTPData(tc.input), "input: %s", tc.input)
	}

	// A large plain-text secret (no embedded comment/CDATA) must still redact:
	// only the comment/CDATA terminator search is windowed, not plain content.
	big := `<password>` + strings.Repeat("S", 40000) + `</password>`
	require.NotContains(t, HTTPData(big), "SSSS", "large plain XML secret leaked")
}

// TestInlinePEMLargeBodyNoTailLeak verifies an inline PEM body larger than the
// END-search window is redacted through the value's closing quote, not truncated
// mid-base64 leaving the tail (and END marker) visible.
func TestInlinePEMLargeBodyNoTailLeak(t *testing.T) {
	t.Parallel()

	for _, body := range []int{16384, 17000, 40000} {
		in := `{"k":"-----BEGIN RSA PRIVATE KEY-----` + strings.Repeat("A", body) + `-----END RSA PRIVATE KEY-----"}`
		out := HTTPData(in)
		require.NotContains(t, out, "AAAA", "inline PEM leaked %d-byte body tail", body)
		require.NotContains(t, out, "END RSA PRIVATE KEY", "inline PEM leaked the END marker for %d-byte body", body)
	}
}

// TestNonSecretKeyAllowlistAllSurfaces verifies keys that tokenize to a sensitive
// keyword but never carry a secret stay visible on every surface (headers, JSON,
// URL), while genuinely sensitive siblings still redact.
//
//nolint:gosec // Field-name fixtures, not real credentials.
func TestNonSecretKeyAllowlistAllSurfaces(t *testing.T) {
	t.Parallel()

	visible := []string{
		`{"securityContext":{"runAsUser":1000}}`,
		`{"security_group":"sg-0a1b2c3d"}`,
		`{"securityGroups":["sg-1"]}`,
		`{"secure":true,"httpOnly":true}`,
		`{"contentSecurityPolicy":"default-src 'self'"}`,
		"Content-Security-Policy: default-src 'self'\n",
		"Strict-Transport-Security: max-age=31536000\n",
		"Access-Control-Allow-Credentials: true\n",
		"WWW-Authenticate: Basic realm=\"api\"\r\n",
		"Proxy-Authenticate: Negotiate\r\n",
		"securityContext=on&user=bob",
	}
	for _, in := range visible {
		require.Equal(t, in, HTTPData(in), "should stay visible: %q", in)
	}

	// Real secrets that share those stems still redact.
	redacts := map[string]string{
		`{"security_answer":"my dog"}`:        `{"security_answer":"***"}`,
		`{"secure_token":"SECRET"}`:           `{"secure_token":"***"}`,
		`{"security_key":"SECRET"}`:           `{"security_key":"***"}`,
		"Authorization: Bearer SEKRIT\n":      "Authorization: ***\n",
		"Proxy-Authorization: Basic SEKRIT\n": "Proxy-Authorization: ***\n",
	}
	for in, want := range redacts {
		require.Equal(t, want, HTTPData(in), "should redact: %q", in)
	}
}

// TestSecureAndAddrTokensDropped verifies the dropped `secure`, `secur` and
// `addr` tokens no longer over-redact ordinary fields, while the postal-address
// token is intentionally kept.
func TestSecureAndAddrTokensDropped(t *testing.T) {
	t.Parallel()

	visible := []string{
		`{"secure":false}`,
		`{"remoteAddr":"203.0.113.7:54321"}`,
		`{"local_addr":"10.0.0.5:8080"}`,
		`remote_addr=203.0.113.7&status=200`,
	}
	for _, in := range visible {
		require.Equal(t, in, HTTPData(in), "should stay visible: %q", in)
	}

	require.Equal(t, expectedRedaction(`{"address":"***"}`), HTTPData(`{"address":"1 Main St"}`))
	require.Equal(t, expectedRedaction(`{"email_addr":"***"}`), HTTPData(`{"email_addr":"a@b.com"}`))
}

// TestTwoWordPairKeys verifies national+id and connection+string match in all
// spellings, without matching lookalikes, alongside the pre-existing
// first/last+name pair.
func TestTwoWordPairKeys(t *testing.T) {
	t.Parallel()

	redacts := []string{
		`{"nationalId":"123-45-6789"}`,
		`{"national_id":"123-45-6789"}`,
		`{"NationalID":"123"}`,
		`{"nationalid":"123"}`,
		`{"connectionString":"Server=db"}`,
		`{"connection_string":"Server=db"}`,
		`{"ConnectionString":"Server=db"}`,
		`{"firstName":"Jane"}`,
		`{"last_name":"Doe"}`,
	}
	for _, in := range redacts {
		require.NotContains(t, HTTPData(in), "123-45-6789", "pair key not redacted: %q", in)
		require.Contains(t, HTTPData(in), "***", "pair key not redacted: %q", in)
	}

	visible := []string{
		`{"international":"yes"}`,
		`{"connectionTimeout":30}`,
		`{"connectionId":"abc"}`,
		`{"idNumber":"x"}`,
	}
	for _, in := range visible {
		require.Equal(t, in, HTTPData(in), "lookalike over-redacted: %q", in)
	}
}

// TestLabeledSecretStrongOnly verifies the {name,value} label rule fires only for
// strong secret labels, not weak/financial/PII ones.
func TestLabeledSecretStrongOnly(t *testing.T) {
	t.Parallel()

	// Weak label values must NOT blank the sibling.
	visible := []string{
		`{"chart":[{"name":"Auth","value":30},{"name":"Cards","value":70}]}`,
		`{"metric":[{"name":"payment_latency_ms","value":42}]}`,
		`{"name":"cardType","value":"visa"}`,
		`{"name":"account","value":"acme"}`,
		`{"name":"email","value":"a@b.com"}`,
	}
	for _, in := range visible {
		require.Equal(t, in, HTTPData(in), "label rule over-redacted: %q", in)
	}

	// Strong label values still redact the sibling.
	cases := []struct {
		input string
		want  string
	}{
		{`{"name":"DB_PASSWORD","value":"hunter2"}`, `{"name":"DB_PASSWORD","value":"***"}`},
		{`{"key":"api_key","value":"AKIA"}`, `{"key":"***","value":"***"}`},
		{`{"name":"clientSecret","value":"x"}`, `{"name":"clientSecret","value":"***"}`},
	}
	for _, tc := range cases {
		require.Equal(t, expectedRedaction(tc.want), HTTPData(tc.input), "input: %s", tc.input)
	}
}

// TestEscapedJSONWhitespace verifies an escaped (embedded) JSON body whose keys
// carry serializer whitespace (Python json.dumps, JS pretty-print) redacts and
// stays convergent, while still bailing safely on inner escaping.
func TestEscapedJSONWhitespace(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  string
	}{
		{`{"body":"{\"a\":\"1\", \"token\":\"hunter2\"}"}`, `{"body":"{\"a\":\"1\", \"token\":\"***\"}"}`},
		{`{"event":"login","body":"{\"user\": \"bob\", \"password\": \"s3cr3t\"}"}`, `{"event":"login","body":"{\"user\": \"bob\", \"password\": \"***\"}"}`},
		{"{\"body\":\"{\\n  \\\"password\\\": \\\"s3cr3t\\\"\\n}\"}", "{\"body\":\"{\\n  \\\"password\\\": \\\"***\\\"\\n}\"}"},
	}
	for _, tc := range cases {
		require.Equal(t, tc.want, HTTPData(tc.input), "input: %s", tc.input)

		twice := HTTPData(HTTPData(tc.input))
		require.Equal(t, twice, HTTPData(twice), "not convergent: %s", tc.input)
	}

	// A backslash-heavy Windows path must not be disturbed (no mis-parse).
	require.Equal(t, expectedRedaction(`{"path":"C:\\Users\\bob\\secret.txt"}`), HTTPData(`{"path":"C:\\Users\\bob\\secret.txt"}`))
}

// TestTracePrefixedHeaders verifies curl/resty per-line trace decorations
// ("> "/"< ") no longer disable the header rule; the prefix is preserved.
func TestTracePrefixedHeaders(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  string
	}{
		{"> Authorization: Bearer SEKRIT", "> Authorization: ***"},
		{"> Proxy-Authorization: Basic SEKRIT", "> Proxy-Authorization: ***"},
		{"  > Authorization: Bearer SEKRIT", "  > Authorization: ***"},
		{"< X-Api-Key: SEKRIT\n", "< X-Api-Key: ***\n"},
		// The trace prefix does not resurrect an ordinary (non-secret) header.
		{"> Host: example.com", "> Host: example.com"},
		// A challenge header stays visible even with a trace prefix.
		{"< WWW-Authenticate: Bearer realm=x", "< WWW-Authenticate: Bearer realm=x"},
	}
	for _, tc := range cases {
		require.Equal(t, expectedRedaction(tc.want), HTTPData(tc.input), "input: %q", tc.input)

		once := HTTPData(tc.input)
		require.Equal(t, once, HTTPData(once), "not idempotent: %q", tc.input)
	}
}

// TestLabeledSecretAttr verifies the HTML name="<secret>" value="..." attribute
// pair redacts the value, in both quote styles, without over-redacting ordinary
// forms.
func TestLabeledSecretAttr(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  string
	}{
		{`<input name="password" value="s3cr3t">`, `<input name="password" value="***">`},
		{`<input name='password' value='s3cr3t'>`, `<input name='password' value='***'>`},
		{`<input name="apikey" value="AKIA0000">`, `<input name="apikey" value="***">`},
		{`<input type="text" name="new_password" value="s3cr3t" id="x">`, `<input type="text" name="new_password" value="***" id="x">`},
		// Ordinary forms are untouched.
		{`<input name="username" value="john">`, `<input name="username" value="john">`},
		{`<input name="password">`, `<input name="password">`},
		{`name=John&city=NYC`, `name=John&city=NYC`},
	}
	for _, tc := range cases {
		require.Equal(t, expectedRedaction(tc.want), HTTPData(tc.input), "input: %q", tc.input)

		once := HTTPData(tc.input)
		require.Equal(t, once, HTTPData(once), "not idempotent: %q", tc.input)
	}
}

// TestLabeledSecretAttrBailPaths exercises the attribute parser's early exits:
// malformed or non-matching shapes must leave the input unchanged.
func TestLabeledSecretAttrBailPaths(t *testing.T) {
	t.Parallel()

	unchanged := []string{
		`<input name="password"value="x">`,      // no whitespace between attributes
		`<input name="password" other="x">`,     // sibling is not a value attribute
		`<input name="password" value x>`,       // sibling has no '='
		`<input name="password" value=x>`,       // sibling value is not quoted
		`<input name="password" value="unterm`,  // sibling value unterminated
		`<input name="password" value="a<b">`,   // value crosses a tag boundary
		`<input name="username" value="john">`,  // label is not a strong secret
		"<input name=\"password\" value=\"a\nb", // value crosses a line boundary
	}
	for _, in := range unchanged {
		require.Equal(t, in, HTTPData(in), "should be unchanged: %q", in)
	}
}

// TestNormalizedPairAndStrongTokens covers the normalized (non-ASCII fallback)
// classification path for the two-word pairs and the strong-secret-token helper
// used by the labeled-secret rule.
func TestNormalizedPairAndStrongTokens(t *testing.T) {
	t.Parallel()

	// Normalized pair path (matchesPairString).
	require.True(t, defaultRedactor.isSensitiveNormalizedKeyTokens("national_id"))
	require.True(t, defaultRedactor.isSensitiveNormalizedKeyTokens("connection_string"))
	require.True(t, defaultRedactor.isSensitiveNormalizedKeyTokens("first_name"))
	require.False(t, defaultRedactor.isSensitiveNormalizedKeyTokens("international"))
	require.False(t, defaultRedactor.isSensitiveNormalizedKeyTokens("connection_timeout"))

	// A key with a non-ASCII byte routes through the normalized fallback path and
	// still matches the pair (the ASCII words national+id are intact).
	require.Equal(t, expectedRedaction(`{"national_id_ä":"***"}`), HTTPData(`{"national_id_ä":"123"}`))

	// isStrongSecretName: strong exact token, glued root suffix, and the drop path.
	require.True(t, defaultRedactor.isStrongSecretName([]byte("api_key")))
	require.True(t, defaultRedactor.isStrongSecretName([]byte("appsecretkey"))) // glued root suffix
	require.False(t, defaultRedactor.isStrongSecretName([]byte("cardType")))    // weak token only

	dropped := New(WithoutTokens("password"))
	require.False(t, dropped.isStrongSecretName([]byte("password")))
	require.Equal(t,
		expectedRedaction(`{"name":"password","value":"keepme"}`),
		dropped.String(`{"name":"password","value":"keepme"}`),
	)
}

// TestEscapedJSONAtInputStart covers the start-of-input branch of the escaped-key
// context scan: an embedded object key at offset 0 is valid key context.
func TestEscapedJSONAtInputStart(t *testing.T) {
	t.Parallel()

	require.Equal(t, `\"password\":\"***\"`, HTTPData(`\"password\":\"secret\"`))
}

// TestVendorSKPrefixNoDash verifies bare sk- keys stop at a dash (so
// dash-separated identifiers / CSS classes stay visible), while the genuinely
// dashed sk-proj-/sk-ant- keys and classic alphanumeric sk- keys still redact.
func TestVendorSKPrefixNoDash(t *testing.T) {
	t.Parallel()

	visible := []string{
		"class sk-loading-skeleton-placeholder-large end",
		"id sk-user-profile-settings-panel-v2 done",
		"the sk-8000 chipset",
	}
	for _, in := range visible {
		require.Equal(t, in, HTTPData(in), "sk- false positive: %q", in)
	}

	redacts := []string{
		"openai sk-proj-AbCdEfGhIjKlMnOpQrStUvWxYz0123456789",
		"anthropic sk-ant-api03-AbCdEfGhIjKlMnOpQrStUv",
		"classic sk-AbCdEfGhIjKlMnOpQrStUvWx012345 done",
	}
	for _, in := range redacts {
		require.Contains(t, HTTPData(in), "***", "sk- key not redacted: %q", in)
	}
}
