package redact

import (
	"strings"
	"sync"
	"unicode"
)

// normalizeKey converts a key name to a tokenized lowercase form suitable for
// boundary-aware keyword checks (e.g., camelCase -> snake_case tokens).
// Uppercase acronym runs followed by a lowercase letter are split before the
// last uppercase letter (e.g., APIKey -> api_key, JWTToken -> jwt_token).
//
//nolint:gocognit,gocyclo,cyclop
func normalizeKey(s string) string {
	var b strings.Builder
	b.Grow(len(s) * 2)

	runes := []rune(s)
	prevIsLowerOrDigit := false
	prevIsUpper := false
	prevIsUnderscore := true

	for i, r := range runes {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			isUpper := unicode.IsUpper(r)
			nextIsLower := i+1 < len(runes) && unicode.IsLower(runes[i+1])

			if isUpper && !prevIsUnderscore && (prevIsLowerOrDigit || (prevIsUpper && nextIsLower)) {
				b.WriteByte('_')
			}

			b.WriteRune(unicode.ToLower(r))

			prevIsLowerOrDigit = unicode.IsLower(r) || unicode.IsDigit(r)
			prevIsUpper = isUpper
			prevIsUnderscore = false

			continue
		}

		if !prevIsUnderscore {
			b.WriteByte('_')
		}

		prevIsLowerOrDigit = false
		prevIsUpper = false
		prevIsUnderscore = true
	}

	return strings.Trim(b.String(), "_")
}

const (
	// sensitiveKeyCacheMaxEntries bounds the key-sensitivity memoization cache.
	// This avoids unbounded memory growth with high-cardinality inputs.
	sensitiveKeyCacheMaxEntries = 4096

	// sensitiveKeyCacheEvictBatch controls how many entries are dropped when the
	// cache is full, reducing lock contention versus evicting one-by-one.
	sensitiveKeyCacheEvictBatch = 512
)

// sensitiveKeyCache memoizes the isSensitive result for keys that take the
// non-ASCII fallback path; pure-ASCII keys (the common case) are classified
// directly by isSensitiveKeyASCIIFast without touching this cache. It avoids
// repeated tokenization/normalization of the same non-ASCII keys.
var sensitiveKeyCache = newSensitiveKeyMemo() //nolint:gochecknoglobals

// Sensitive tokens are precomputed once and matched against normalized key tokens.
var sensitiveTokens = map[string]struct{}{ //nolint:gochecknoglobals
	"acc":              {},
	"accesskey":        {},
	"accesstoken":      {},
	"account":          {},
	"addr":             {},
	"address":          {},
	"amount":           {},
	"apikey":           {},
	"apisecret":        {},
	"appsecret":        {},
	"attestation":      {},
	"auth":             {},
	"authenticate":     {},
	"authorization":    {},
	"authtoken":        {},
	"autograph":        {},
	"bal":              {},
	"balance":          {},
	"bearer":           {},
	"bearertoken":      {},
	"bill":             {},
	"birth":            {},
	"card":             {},
	"cc":               {},
	"cell":             {},
	"cert":             {},
	"checksum":         {},
	"clientsecret":     {},
	"connectionstring": {},
	"cookie":           {},
	"cred":             {},
	"credential":       {},
	"credentials":      {},
	"csrf":             {},
	"csrftoken":        {},
	"cv2":              {},
	"cvc":              {},
	"cvv":              {},
	"dbpassword":       {},
	"dob":              {},
	"dsa":              {},
	"dsn":              {},
	"ecdsa":            {},
	"email":            {},
	"endorse":          {},
	"expiry":           {},
	"fingerprint":      {},
	"hash":             {},
	"hmac":             {},
	"holder":           {},
	"htpasswd":         {},
	"iban":             {},
	"idtoken":          {},
	"jsessionid":       {},
	"jwt":              {},
	"key":              {},
	"login":            {},
	"mail":             {},
	"mfa":              {},
	"nonce":            {},
	"otp":              {},
	"pass":             {},
	"passcode":         {},
	"passphrase":       {},
	"password":         {},
	"pay":              {},
	"payment":          {},
	"pgpassword":       {},
	"phone":            {},
	"phpsessid":        {},
	"pin":              {},
	"pkcs":             {},
	"pkcs12":           {},
	"postal":           {},
	"privatekey":       {},
	"proof":            {},
	"pwd":              {},
	"refreshtoken":     {},
	"rsa":              {},
	"salt":             {},
	"seal":             {},
	"secret":           {},
	"secretkey":        {},
	"secur":            {},
	"secure":           {},
	"security":         {},
	"sess":             {},
	"session":          {},
	"sessionid":        {},
	"sessionkey":       {},
	"sessiontoken":     {},
	"sgn":              {},
	"sid":              {},
	"sig":              {},
	"signature":        {},
	"social":           {},
	"ssn":              {},
	"swift":            {},
	"tax":              {},
	"tel":              {},
	"telephone":        {},
	"token":            {},
	"totp":             {},
	"xsrf":             {},
	"xsrftoken":        {},
}

// sensitiveKeyMemo is a small concurrent memoization cache for sensitive-key checks.
type sensitiveKeyMemo struct {
	mu   sync.RWMutex
	data map[string]bool
}

// newSensitiveKeyMemo creates a bounded memo cache pre-sized for max entries.
func newSensitiveKeyMemo() *sensitiveKeyMemo {
	return &sensitiveKeyMemo{data: make(map[string]bool, sensitiveKeyCacheMaxEntries)}
}

// get performs a read-locked lookup.
func (c *sensitiveKeyMemo) get(k string) (bool, bool) {
	c.mu.RLock()
	v, ok := c.data[k]
	c.mu.RUnlock()

	return v, ok
}

// set inserts or updates a memoized key.
// When full, it evicts a small batch of arbitrary entries to keep memory bounded.
func (c *sensitiveKeyMemo) set(k string, v bool) {
	c.mu.Lock()
	if len(c.data) >= sensitiveKeyCacheMaxEntries {
		evicted := 0

		for key := range c.data {
			delete(c.data, key)

			evicted++
			if evicted >= sensitiveKeyCacheEvictBatch {
				break
			}
		}
	}

	c.data[k] = v
	c.mu.Unlock()
}

// isSensitiveKey reports whether a key contains a sensitive token after
// normalization, honoring the instance's extra and dropped tokens.
func (re *Redactor) isSensitiveKey(key []byte) bool {
	if result, ok := re.sensitiveKeyASCIIFast(key); ok {
		return result
	}

	k := string(key)
	if v, ok := re.keyMemo.get(k); ok {
		return v
	}

	result := re.isSensitiveNormalizedKeyTokens(normalizeKey(k))
	re.keyMemo.set(k, result)

	return result
}

//nolint:cyclop,gocognit,gocyclo
func (re *Redactor) sensitiveKeyASCIIFast(key []byte) (bool, bool) {
	start := -1
	prevIsLowerOrDigit := false
	prevIsUpper := false
	prevIsUnderscore := true
	prevFirstOrLast := false

	for i := range key {
		c := key[i]

		if c >= 0x80 {
			return false, false
		}

		if isASCIIAlphaNum(c) {
			isUpper := c >= 'A' && c <= 'Z'
			isLower := c >= 'a' && c <= 'z'

			switch {
			case isUpper && prevIsLowerOrDigit && !prevIsUnderscore && start >= 0:
				// camelCase boundary: a lower/digit-to-upper transition closes the token.
				sensitive, firstOrLast := re.closeSensitiveToken(key[start:i], prevFirstOrLast)
				if sensitive {
					return true, true
				}

				prevFirstOrLast = firstOrLast
				start = i
			case isLower && prevIsUpper && start >= 0 && i-1 > start:
				// Acronym-run boundary (e.g. APIKey, JWTToken, CCNumber): the last
				// uppercase letter starts the next word, so the token ends before it.
				sensitive, firstOrLast := re.closeSensitiveToken(key[start:i-1], prevFirstOrLast)
				if sensitive {
					return true, true
				}

				prevFirstOrLast = firstOrLast
				start = i - 1
			case start < 0:
				start = i
			}

			prevIsUpper = isUpper
			prevIsLowerOrDigit = isLower || (c >= '0' && c <= '9')
			prevIsUnderscore = false

			continue
		}

		if start >= 0 {
			sensitive, firstOrLast := re.closeSensitiveToken(key[start:i], prevFirstOrLast)
			if sensitive {
				return true, true
			}

			prevFirstOrLast = firstOrLast
			start = -1
		}

		prevIsUpper = false
		prevIsLowerOrDigit = false
		prevIsUnderscore = true
	}

	if start >= 0 {
		sensitive, _ := re.closeSensitiveToken(key[start:], prevFirstOrLast)
		if sensitive {
			return true, true
		}
	}

	return false, true
}

// closeSensitiveToken checks a completed token: it reports whether the token
// (or the "first"/"last" + "name" pair) is sensitive, and whether the token
// primes the first/last-name special case for the next token.
func (re *Redactor) closeSensitiveToken(tok []byte, prevFirstOrLast bool) (bool, bool) {
	if re.sensitiveTokenASCII(tok) {
		return true, false
	}

	if prevFirstOrLast && equalsASCIIFold(tok, "name") {
		return true, false
	}

	return false, equalsASCIIFold(tok, "first") || equalsASCIIFold(tok, "last")
}

func isASCIIAlphaNum(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

// maxSensitiveTokenLen is an upper bound on the length of any sensitiveTokens
// key ("authorization" is the longest at 13 bytes). Longer candidates cannot
// match and are rejected before the lookup; TestSensitiveTokenLengths guards
// the bound when new tokens are added.
const maxSensitiveTokenLen = 16

// sensitiveTokenASCII reports whether tok is a sensitive keyword,
// case-insensitively. It lowercases into a fixed stack buffer and relies on
// the compiler's allocation-free map[string(bytes)] lookup optimization, so
// sensitiveTokens stays the single source of truth for the built-in set,
// adjusted by the instance's extra and dropped tokens.
func (re *Redactor) sensitiveTokenASCII(tok []byte) bool {
	if len(tok) == 0 || len(tok) > maxSensitiveTokenLen {
		return false
	}

	var buf [maxSensitiveTokenLen]byte
	for i, c := range tok {
		buf[i] = lowerASCIIByte(c)
	}

	return re.sensitiveToken(string(buf[:len(tok)]))
}

// sensitiveToken checks a lowercased token against the built-in set adjusted
// by the instance configuration.
func (re *Redactor) sensitiveToken(lowered string) bool {
	if re.dropTokens != nil {
		if _, dropped := re.dropTokens[lowered]; dropped {
			return false
		}
	}

	if _, ok := sensitiveTokens[lowered]; ok {
		return true
	}

	if re.extraTokens != nil {
		if _, ok := re.extraTokens[lowered]; ok {
			return true
		}
	}

	return false
}

func lowerASCIIByte(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + ('a' - 'A')
	}

	return c
}

func equalsASCIIFold(tok []byte, lit string) bool {
	if len(tok) != len(lit) {
		return false
	}

	for i := range tok {
		c := tok[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}

		if c != lit[i] {
			return false
		}
	}

	return true
}

//nolint:cyclop,gocognit,gocyclo
func (re *Redactor) isSensitiveNormalizedKeyTokens(normalized string) bool {
	if normalized == "" {
		return false
	}

	start := 0
	prevStart := -1
	prevEnd := -1

	for i := 0; i <= len(normalized); i++ {
		if i < len(normalized) && normalized[i] != '_' {
			continue
		}

		if i > start {
			tok := normalized[start:i]
			if re.sensitiveToken(tok) {
				return true
			}

			if prevStart >= 0 {
				prev := normalized[prevStart:prevEnd]
				if (prev == "first" || prev == "last") && tok == "name" {
					return true
				}
			}

			prevStart = start
			prevEnd = i
		}

		start = i + 1
	}

	return false
}
