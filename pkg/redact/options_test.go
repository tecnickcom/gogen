package redact

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWithMarker(t *testing.T) {
	t.Parallel()

	re := New(WithMarker("#REDACTED#"))

	cases := []struct {
		input string
		want  string
	}{
		{"Authorization: Bearer SECRET\n", "Authorization: #REDACTED#\n"},
		{`{"password":"x"}`, `{"password":"#REDACTED#"}`},
		{"token=SECRET&note=ok", "token=#REDACTED#&note=ok"},
		{"4012888888881881", "#REDACTED#"},
		{"state=eyJhbGciOiJIUzI1NiJ9.abc.def", "state=#REDACTED#"},
		{"<password>SECRET</password>", "<password>#REDACTED#</password>"},
		{"postgres://u:p@h", "postgres://u:#REDACTED#@h"},
	}

	for _, tc := range cases {
		require.Equal(t, tc.want, re.String(tc.input), "input: %s", tc.input)

		// Custom markers keep single-pass stability on realistic input.
		once := re.String(tc.input)
		require.Equal(t, once, re.String(once), "not idempotent for input: %s", tc.input)
	}

	// An empty marker is ignored and keeps the default.
	require.Equal(t, "token=***", New(WithMarker("")).String("token=SECRET"))
}

// TestWithLuhnCheckIsInstanceScoped verifies the Luhn gate is per-instance:
// one Redactor enabling it changes nothing for other instances or for the
// package-level API, which always runs with the gate off.
func TestWithLuhnCheckIsInstanceScoped(t *testing.T) {
	t.Parallel()

	invalidLuhn := "4012888888881882" // valid Visa prefix/length, invalid checksum.

	strict := New(WithLuhnCheck(true))
	relaxed := New()
	explicitOff := New(WithLuhnCheck(false))

	// The strict instance rejects the invalid checksum; the other instances and
	// the package default (gate off) redact on prefix alone.
	require.Equal(t, invalidLuhn, strict.String(invalidLuhn))
	require.Equal(t, RedactionMarker, relaxed.String(invalidLuhn))
	require.Equal(t, RedactionMarker, explicitOff.String(invalidLuhn))
	require.Equal(t, RedactionMarker, String(invalidLuhn))

	// A Luhn-valid card is redacted everywhere.
	validLuhn := "4012888888881881"
	require.Equal(t, RedactionMarker, strict.String(validLuhn))
	require.Equal(t, RedactionMarker, String(validLuhn))
}

func TestWithExtraTokens(t *testing.T) {
	t.Parallel()

	re := New(WithExtraTokens("floof", "WOBBLE"))

	cases := []struct {
		input string
		want  string
	}{
		{"floof=SECRET", "floof=***"},
		{`{"floof":"SECRET"}`, `{"floof":"***"}`},
		{`{"userFloof":"SECRET"}`, `{"userFloof":"***"}`}, // tokenized match
		{"Floof: SECRET\n", "Floof: ***\n"},               // header rule
		{"wobble=SECRET", "wobble=***"},                   // lowercased at construction
		{"floofy=VISIBLE", "floofy=VISIBLE"},              // exact-token, not substring
	}

	for _, tc := range cases {
		require.Equal(t, tc.want, re.String(tc.input), "input: %s", tc.input)
	}

	// The default instance is unaffected.
	require.Equal(t, "floof=SECRET", String("floof=SECRET"))

	// Unusable tokens are ignored: empty, non-alphanumeric, too long.
	loose := New(WithExtraTokens("", "api-key", strings.Repeat("a", maxSensitiveTokenLen+1)))
	require.Equal(t, "apikeyx=VISIBLE", loose.String("apikeyx=VISIBLE"))
}

func TestWithoutTokens(t *testing.T) {
	t.Parallel()

	re := New(WithoutTokens("amount", "Balance"))

	cases := []struct {
		input string
		want  string
	}{
		{`{"amount": 9999}`, `{"amount": 9999}`},
		{"amount_due=100", "amount_due=100"},
		{`{"balance":-1.5}`, `{"balance":-1.5}`},
		// Other tokens keep redacting on the same instance.
		{`{"password":"x"}`, `{"password":"***"}`},
	}

	for _, tc := range cases {
		require.Equal(t, tc.want, re.String(tc.input), "input: %s", tc.input)
	}

	// The default instance still redacts the dropped tokens.
	require.Equal(t, `{"amount": "***"}`, String(`{"amount": 9999}`)) //nolint:testifylint // byte-exact output comparison is the point
}

// TestWithoutTokensDropsRootSuffixRule verifies that dropping a token that is
// also a glued-compound root disables its suffix rule and its plural, so the
// three matching modes stay coherent.
func TestWithoutTokensDropsRootSuffixRule(t *testing.T) {
	t.Parallel()

	re := New(WithoutTokens("secret"))

	cases := []struct {
		input string
		want  string
	}{
		{"secret=VISIBLE", "secret=VISIBLE"},        // exact
		{"mysecret=VISIBLE", "mysecret=VISIBLE"},    // root suffix
		{"secrets=VISIBLE", "secrets=VISIBLE"},      // plural
		{"my_secret=VISIBLE", "my_secret=VISIBLE"},  // tokenized
		{"clientsecret=SECRET", "clientsecret=***"}, // separate enumerated token
		{"password=SECRET", "password=***"},         // other roots keep matching
		{"newpassword=SECRET", "newpassword=***"},   // other roots keep matching
	}

	for _, tc := range cases {
		require.Equal(t, expectedRedaction(tc.want), re.String(tc.input), "input: %s", tc.input)
	}

	// The default instance is unaffected.
	require.Equal(t, "mysecret=***", String("mysecret=SECRET"))
}

//nolint:gosec // The PEM fixtures are fake fragments, not real credentials.
func TestWithoutRules(t *testing.T) {
	t.Parallel()

	cases := []struct {
		rule  Rule
		input string
		want  string // output of the default engine, for contrast
	}{
		{RuleHeaders, "Authorization: Bearer SECRET\n", "Authorization: ***\n"},
		{RuleJSON, `{"password":"x"}`, `{"password":"***"}`},
		{RuleURLEncoded, "token=SECRET", "token=***"},
		{RuleXML, "<password>SECRET</password>", "<password>***</password>"},
		{RuleUserinfo, "postgres://u:p@h", "postgres://u:***@h"},
		{RuleJWT, "eyJhbGciOiJIUzI1NiJ9.abc.def", "***"},
		{RuleVendorTokens, "ghp_AbCdEfGhIjKlMnOpQrStUvWxYz012345", "***"},
		{RulePEM, "-----BEGIN RSA PRIVATE KEY-----\nMIIE\n-----END RSA PRIVATE KEY-----", "-----BEGIN RSA PRIVATE KEY-----\n***\n-----END RSA PRIVATE KEY-----"},
		{RuleCards, "4012888888881881", "***"},
		{RuleCards, "4012 8888 8888 1881", "***"},
	}

	for _, tc := range cases {
		// Disabled: input passes through untouched.
		re := New(WithoutRules(tc.rule))
		require.Equal(t, tc.input, re.String(tc.input), "rule %b disabled, input: %s", tc.rule, tc.input)

		// Enabled elsewhere: the default engine redacts it.
		require.Equal(t, expectedRedaction(tc.want), String(tc.input), "default engine, input: %s", tc.input)
	}

	// Combined rule sets disable together.
	re := New(WithoutRules(RuleJSON, RuleURLEncoded))
	require.Equal(t, `{"password":"x"}`, re.String(`{"password":"x"}`)) //nolint:testifylint // byte-exact output comparison is the point
	require.Equal(t, "token=SECRET", re.String("token=SECRET"))
	require.Equal(t, "Authorization: ***\n", re.String("Authorization: Bearer S\n"))

	// Inline PEM is governed by RulePEM too.
	inline := `{"data":"-----BEGIN RSA PRIVATE KEY-----\nMIIE\n-----END RSA PRIVATE KEY-----"}`
	require.Equal(t, inline, New(WithoutRules(RulePEM)).String(inline))
}

func TestRedactorNilOptionIsIgnored(t *testing.T) {
	t.Parallel()

	require.NotPanics(t, func() {
		re := New(nil, WithMarker("X"))
		require.Equal(t, "token=X", re.String("token=SECRET"))
	})
}
