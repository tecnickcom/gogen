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

// WithLuhnCheck sets the instance-scoped Luhn-checksum gate for credit-card
// detection (see [SetLuhnCheck] for the semantics). Unlike the package-level
// toggle, the instance flag is fixed at construction and unaffected by
// SetLuhnCheck.
func WithLuhnCheck(enabled bool) Option {
	return func(re *Redactor) {
		re.luhn.Store(enabled)
	}
}

// WithExtraTokens adds sensitive key tokens to the built-in set. Tokens are
// matched case-insensitively against the tokenized key form (camelCase,
// snake_case, kebab-case and acronym runs all split into tokens), exactly
// like the built-in keywords. Tokens must be plain ASCII letters and digits
// and at most 16 bytes after lowercasing; anything else is ignored.
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
// readable in financial logs). The same token constraints as
// [WithExtraTokens] apply.
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

// normalizeConfigToken lowercases a configured token and reports whether it
// is usable: non-empty, at most maxSensitiveTokenLen bytes, plain ASCII
// letters and digits (anything else can never match a tokenized key).
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
