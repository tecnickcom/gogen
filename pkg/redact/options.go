package redact

// Rule identifies one redaction rule class of the engine. Rule values are
// bit flags and can be combined with bitwise OR.
type Rule uint16

// Rule classes accepted by [WithoutRules].
const (
	// RuleHeaders redacts the values of HTTP headers with sensitive names.
	RuleHeaders Rule = 1 << iota
	// RuleJSON redacts JSON values whose key name is sensitive.
	RuleJSON
	// RuleURLEncoded redacts URL-encoded (key=value) pairs with sensitive keys.
	RuleURLEncoded
	// RuleXML redacts XML element content for elements with sensitive names.
	RuleXML
	// RuleUserinfo redacts passwords in URL userinfo (scheme://user:pass@host).
	RuleUserinfo
	// RuleJWT redacts JWT/JWE compact tokens (eyJ...).
	RuleJWT
	// RuleVendorTokens redacts vendor credential literals (ghp_, sk-, AKIA, ...).
	RuleVendorTokens
	// RulePEM redacts PEM private-key block bodies.
	RulePEM
	// RuleCards redacts credit-card numbers (contiguous and grouped).
	RuleCards
)

// Option configures a [Redactor] during [New].
type Option func(*Redactor)

// WithMarker replaces the redaction marker (default `***`).
//
// The marker must be inert text: it should not contain double quotes or
// structural bytes (braces, brackets, '=', ':') and must not itself look like
// a secret (e.g. a vendor token prefix), or the engine's convergence
// guarantees degrade. An empty marker is ignored.
func WithMarker(marker string) Option {
	return func(re *Redactor) {
		if marker != "" {
			re.marker = []byte(marker)
		}
	}
}

// WithLuhnCheck enables the Luhn-checksum gate for credit-card detection.
//
// It is off by default: every 13-19 digit run matching a known card prefix is
// redacted (a deliberate, safe over-redaction). When enabled, only digit runs
// that match a known prefix AND pass the Luhn checksum are redacted, which
// reduces over-redaction of unrelated numeric identifiers at the cost of
// possibly missing malformed numbers. Enabling it also unlocks detection of
// short (12-15 digit) Maestro numbers, which are too collision-prone to detect
// on prefix alone.
//
// The gate is instance-scoped and fixed at construction.
func WithLuhnCheck(enabled bool) Option {
	return func(re *Redactor) {
		re.luhn = enabled
	}
}

// WithExtraTokens adds sensitive key tokens to the built-in set. Each argument
// must be a SINGLE word token: it is matched case-insensitively against the
// tokenized key form (camelCase, snake_case, kebab-case and acronym runs all
// split into tokens), so an extra token "floof" redacts keys named `floof`,
// `userFloof`, `floof_id` and (as a numbered field) `floof2`. Extra tokens are
// matched as whole tokens only — unlike the built-in roots (`password`,
// `token`, ...) they are NOT matched as the tail of a longer glued compound
// (`myfloof`) nor as a bare plural (`floofs`).
//
// Each argument must be plain ASCII letters/digits, non-empty, and at most 32
// bytes; anything else (including a name with separators such as `api_key`,
// `x-api-key`, or `session-id`) is SILENTLY IGNORED. Pass the distinctive words
// of a multi-word field separately — e.g. WithExtraTokens("tenant", "vault") —
// bearing in mind each becomes a standalone token. Most multi-word secret names
// already redact because one component (`key`, `token`, `secret`, ...) is a
// built-in keyword, so an extra token is only needed for a fully house-style word.
func WithExtraTokens(tokens ...string) Option {
	return func(re *Redactor) {
		for _, tok := range tokens {
			if lowered, ok := normalizeConfigToken(tok); ok {
				if re.extraTokens == nil {
					re.extraTokens = make(map[string]struct{}, len(tokens))
				}

				re.extraTokens[lowered] = struct{}{}
			}
		}
	}
}

// WithoutTokens removes tokens from the built-in sensitive set, so keys
// matching only those tokens stay visible (e.g. keep `amount`/`balance`
// readable in financial logs). Dropping a token disables all of its matching
// forms together — exact, numbered (`secret2`), plural (`secrets`), and the
// glued-compound suffix (`mysecret`) — so WithoutTokens("secret") keeps every
// one of them visible; other enumerated tokens that happen to end in it
// (`clientsecret`) are separate entries and must be dropped by name. The same
// single-word token constraints as [WithExtraTokens] apply.
func WithoutTokens(tokens ...string) Option {
	return func(re *Redactor) {
		for _, tok := range tokens {
			if lowered, ok := normalizeConfigToken(tok); ok {
				if re.dropTokens == nil {
					re.dropTokens = make(map[string]struct{}, len(tokens))
				}

				re.dropTokens[lowered] = struct{}{}
			}
		}
	}
}

// WithoutRules disables entire rule classes on the instance.
func WithoutRules(rules ...Rule) Option {
	return func(re *Redactor) {
		for _, r := range rules {
			re.disabled |= r
		}
	}
}

// normalizeConfigToken lowercases a configured token and reports whether it is
// usable: non-empty, at most maxSensitiveTokenLen bytes, plain ASCII letters
// and digits (anything else can never match a tokenized key).
func normalizeConfigToken(tok string) (string, bool) {
	if tok == "" || len(tok) > maxSensitiveTokenLen {
		return "", false
	}

	var buf [maxSensitiveTokenLen]byte

	for i := range len(tok) {
		c := lowerASCIIByte(tok[i])
		if !isASCIIAlphaNum(c) {
			return "", false
		}

		buf[i] = c
	}

	return string(buf[:len(tok)]), true
}
