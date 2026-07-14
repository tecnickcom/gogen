package redact

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRedactVendorTokens(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  string
	}{
		// Vendor credentials in free text are replaced whole, prefix included.
		{"push failed: ghp_AbCdEfGhIjKlMnOpQrStUvWxYz012345 rejected", "push failed: *** rejected"},
		{"github_pat_11ABCDEFG0123456789abcdef", "***"},
		{"gho_AbCdEfGhIjKlMnOpQrStUvWxYz012345", "***"},
		{"ghs_AbCdEfGhIjKlMnOpQrStUvWxYz012345", "***"},
		{"slack xoxb-1234567890-abcdefghij x", "slack *** x"},
		{"xoxp-1234567890-abcdefghij", "***"},
		{"stripe sk_live_4eC39HqLyjWDarjtT1zdp7dc", "stripe ***"},
		{"sk_test_4eC39HqLyjWDarjtT1zdp7dc", "***"},
		{"rk_live_4eC39HqLyjWDarjtT1zdp7dc", "***"},
		{"whsec_abcdefghijklmnop123456", "***"},
		{"aws AKIAIOSFODNN7EXAMPLE used", "aws *** used"},
		{"ASIAIOSFODNN7EXAMPLE", "***"},
		{"google AIzaSyD-1234567890abcdefghijklmnop", "google ***"},
		{"gitlab glpat-xxxxxxxxxxxxxxxxxxxx", "gitlab ***"},
		{"dop_v1_0123456789abcdef0123456789abcdef", "***"},
		{"shpat_0123456789abcdef0123456789abcdef", "***"},
		// OpenAI / Anthropic / Hugging Face / SendGrid / Docker Hub.
		{"openai sk-proj-AbCdEfGhIjKlMnOpQrStUvWxYz0123456789", "openai ***"},
		{"anthropic sk-ant-api03-AbCdEfGhIjKlMnOpQrStUv", "anthropic ***"},
		{"huggingface hf_AbCdEfGhIjKlMnOpQrStUvWxYz", "huggingface ***"},
		{"sendgrid SG.AbCdEfGhIjKlMnOp.QrStUvWxYz0123456789abcdef", "sendgrid ***"},
		{"docker dckr_pat_AbCdEfGhIjKlMnOpQrStUvWx", "docker ***"},
		// Sentence punctuation after a dot-body token is preserved.
		{"key SG.AbCdEfGhIjKlMnOp.QrStUvWxYz01234. Next", "key ***. Next"},
		// Inside JSON string values and URL-encoded pairs.
		{`{"note":"ghp_AbCdEfGhIjKlMnOpQrStUvWxYz012345"}`, `{"note":"***"}`},
		{"ref=ghp_AbCdEfGhIjKlMnOpQrStUvWxYz012345", "ref=***"},
		// Prose containing the prescreen bigrams stays visible.
		{"we should skip what github.com does", "we should skip what github.com does"},
		{"the gift shop wholesale doprava", "the gift shop wholesale doprava"},
		{"the sk-8000 chipset", "the sk-8000 chipset"},
		{"hf_radio", "hf_radio"},
		{"SG. is a country code", "SG. is a country code"},
		{"a dckr_pat x", "a dckr_pat x"},
		// Tail too short: not a credential.
		{"ghp_short x", "ghp_short x"},
		{"xoxb-12 x", "xoxb-12 x"},
		// Glued to a preceding word character: an identifier.
		{"Aghp_AbCdEfGhIjKlMnOpQrStUvWx", "Aghp_AbCdEfGhIjKlMnOpQrStUvWx"},
		// Candidate byte at end of input.
		{"a g", "a g"},
	}

	for _, tc := range cases {
		require.Equal(t, expectedRedaction(tc.want), Default().String(tc.input), "input: %s", tc.input)
	}
}

// TestVendorTokenPrefixTable guards consistency between the prefix table, the
// first-byte dispatch, and the three-byte prescreen: every configured prefix
// must be reachable through both gates, and a matching token must redact.
func TestVendorTokenPrefixTable(t *testing.T) {
	t.Parallel()

	// Non-candidate bytes are rejected by the prescreen.
	require.False(t, isVendorTokenStart([]byte("zzz"), 0))
	require.False(t, isVendorTokenStart([]byte("gh"), 0)) // too short

	for first, prefixes := range vendorTokenPrefixes {
		require.Equalf(t, trigVendor, bulkTrigger[first],
			"first byte %q not classified as trigVendor in bulkTrigger", first)

		for _, vp := range prefixes {
			require.Equalf(t, first, vp.prefix[0],
				"prefix %q filed under wrong first byte %q", vp.prefix, first)

			require.Truef(t, isVendorTokenStart([]byte(vp.prefix), 0),
				"prefix %q rejected by the isVendorTokenStart prescreen", vp.prefix)

			// A synthetic token with the minimum tail must redact end-to-end.
			token := vp.prefix + strings.Repeat("0", vp.minTail)
			require.Equalf(t, RedactionMarker, Default().String(token),
				"synthetic token for prefix %q not redacted", vp.prefix)
		}
	}
}
