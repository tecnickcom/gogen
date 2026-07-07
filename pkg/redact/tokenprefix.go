package redact

// vendorPrefix describes one well-known credential prefix: tokens are matched
// as prefix + at least minTail token-body characters at a word boundary.
// dotBody additionally allows '.' inside the token body (SendGrid keys use
// dot-joined segments); trailing dots are trimmed so sentence punctuation
// after a token is preserved.
type vendorPrefix struct {
	prefix  string
	minTail int
	dotBody bool
}

// vendorTokenPrefixes maps a token's first byte to the candidate prefixes
// starting with it. Vendors chose these prefixes to be unmistakable, so
// matching them is precise (the same philosophy as the "eyJ" JWT rule). The
// whole token, prefix included, is replaced by the marker: the prefix alone
// would still reveal which vendor's credential leaked.
var vendorTokenPrefixes = map[byte][]vendorPrefix{ //nolint:gochecknoglobals
	'g': {
		{prefix: "github_pat_", minTail: 20}, // GitHub fine-grained PAT
		{prefix: "ghp_", minTail: 20},        // GitHub classic PAT
		{prefix: "gho_", minTail: 20},        // GitHub OAuth token
		{prefix: "ghu_", minTail: 20},        // GitHub user-to-server token
		{prefix: "ghs_", minTail: 20},        // GitHub server-to-server token
		{prefix: "ghr_", minTail: 20},        // GitHub refresh token
		{prefix: "glpat-", minTail: 16},      // GitLab PAT
	},
	'x': {
		{prefix: "xoxb-", minTail: 10}, // Slack bot token
		{prefix: "xoxp-", minTail: 10}, // Slack user token
		{prefix: "xoxa-", minTail: 10}, // Slack app token
		{prefix: "xoxs-", minTail: 10}, // Slack session token
		{prefix: "xoxe-", minTail: 10}, // Slack token rotation
	},
	's': {
		{prefix: "sk_live_", minTail: 16}, // Stripe live secret key
		{prefix: "sk_test_", minTail: 16}, // Stripe test secret key
		{prefix: "sk-", minTail: 20},      // OpenAI / Anthropic API keys (sk-proj-, sk-ant-, ...)
		{prefix: "shpat_", minTail: 24},   // Shopify admin token
		{prefix: "shpss_", minTail: 24},   // Shopify shared secret
	},
	'S': {
		{prefix: "SG.", minTail: 16, dotBody: true}, // SendGrid API key
	},
	'h': {
		{prefix: "hf_", minTail: 20}, // Hugging Face token
	},
	'r': {
		{prefix: "rk_live_", minTail: 16}, // Stripe live restricted key
		{prefix: "rk_test_", minTail: 16}, // Stripe test restricted key
	},
	'w': {
		{prefix: "whsec_", minTail: 16}, // Stripe webhook signing secret
	},
	'd': {
		{prefix: "dop_v1_", minTail: 32},   // DigitalOcean PAT
		{prefix: "dckr_pat_", minTail: 20}, // Docker Hub PAT
	},
	'A': {
		{prefix: "AKIA", minTail: 16}, // AWS access key id
		{prefix: "ASIA", minTail: 16}, // AWS temporary access key id
		{prefix: "AIza", minTail: 30}, // Google API key
	},
}

// isVendorTokenStart is the cheap three-byte prescreen used by the bulk-copy
// loop: it must be true for every vendorTokenPrefixes entry while rejecting
// nearly all ordinary words ("skip", "what", "should", ...) so the hot loop
// stays tight. The switch on a three-byte string conversion compiles without
// allocating. TestVendorTokenPrefixTable guards consistency with the table.
func isVendorTokenStart(src []byte, i int) bool {
	if i+2 >= len(src) {
		return false
	}

	switch string(src[i : i+3]) {
	case "ghp", "gho", "ghu", "ghs", "ghr", "git", "glp",
		"xox", "sk_", "sk-", "shp", "SG.", "hf_", "rk_", "whs", "dop", "dck",
		"AKI", "ASI", "AIz":
		return true
	default:
		return false
	}
}

// appendRedactedVendorTokenAt handles a candidate byte at src[i]: when a
// well-known vendor credential prefix starts at a word boundary and is
// followed by a long enough token body, the whole token is replaced with the
// marker and the index just past it is returned.
func (re *Redactor) appendRedactedVendorTokenAt(src []byte, i int, dst []byte) (int, []byte, bool) {
	if i > 0 && isWordChar(src[i-1]) {
		return 0, dst, false
	}

	for _, vp := range vendorTokenPrefixes[src[i]] {
		if !hasPrefixAt(src, i, vp.prefix) {
			continue
		}

		end := scanVendorTokenBody(src, i+len(vp.prefix), vp.dotBody)
		if end-(i+len(vp.prefix)) < vp.minTail {
			continue
		}

		dst = append(dst, re.marker...)

		return end, dst, true
	}

	return 0, dst, false
}

// scanVendorTokenBody returns the index just past the run of token-body bytes
// (word characters and '-', plus '.' when dotBody is set) starting at i.
// Trailing dots are trimmed so sentence punctuation after a token is
// preserved.
func scanVendorTokenBody(src []byte, i int, dotBody bool) int {
	for i < len(src) && (isWordChar(src[i]) || src[i] == '-' || (dotBody && src[i] == '.')) {
		i++
	}

	for dotBody && i > 0 && src[i-1] == '.' {
		i--
	}

	return i
}
