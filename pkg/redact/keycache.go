package redact

import (
	"sync"
)

const (
	// sensitiveKeyCacheMaxEntries bounds the key-sensitivity memoization cache.
	// This avoids unbounded memory growth with high-cardinality inputs.
	sensitiveKeyCacheMaxEntries = 4096

	// sensitiveKeyCacheEvictBatch controls how many entries are dropped when the
	// cache is full, reducing lock contention versus evicting one-by-one.
	sensitiveKeyCacheEvictBatch = 512
)

// sensitiveKeyCache maps raw key strings to their isSensitive result, avoiding
// repeated normalisation and regex work for keys that appear on every request.
var sensitiveKeyCache = newSensitiveKeyMemo() //nolint:gochecknoglobals

// Sensitive tokens are precomputed once and matched against normalized key tokens.
var sensitiveTokens = map[string]struct{}{ //nolint:gochecknoglobals
	"acc":         {},
	"account":     {},
	"addr":        {},
	"address":     {},
	"amount":      {},
	"attestation": {},
	"auth":        {},
	"autograph":   {},
	"bal":         {},
	"balance":     {},
	"bearer":      {},
	"bill":        {},
	"birth":       {},
	"card":        {},
	"cc":          {},
	"cell":        {},
	"cert":        {},
	"checksum":    {},
	"cookie":      {},
	"cred":        {},
	"cv2":         {},
	"cvc":         {},
	"cvv":         {},
	"dob":         {},
	"dsa":         {},
	"dsn":         {},
	"ecdsa":       {},
	"email":       {},
	"endorse":     {},
	"expiry":      {},
	"fingerprint": {},
	"hash":        {},
	"hmac":        {},
	"holder":      {},
	"iban":        {},
	"jwt":         {},
	"key":         {},
	"login":       {},
	"mail":        {},
	"pass":        {},
	"password":    {},
	"pay":         {},
	"payment":     {},
	"phone":       {},
	"pkcs":        {},
	"pkcs12":      {},
	"postal":      {},
	"proof":       {},
	"pwd":         {},
	"rsa":         {},
	"salt":        {},
	"seal":        {},
	"secret":      {},
	"secur":       {},
	"secure":      {},
	"security":    {},
	"sess":        {},
	"session":     {},
	"sgn":         {},
	"sid":         {},
	"sig":         {},
	"social":      {},
	"ssn":         {},
	"swift":       {},
	"tax":         {},
	"tel":         {},
	"telephone":   {},
	"token":       {},
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

// isSensitiveKey reports whether a key contains a sensitive token after normalization.
func isSensitiveKeyBytes(key []byte) bool {
	if result, ok := isSensitiveKeyASCIIFast(key); ok {
		return result
	}

	k := string(key)
	if v, ok := sensitiveKeyCache.get(k); ok {
		return v
	}

	result := isSensitiveNormalizedKey(normalizeKey(k))
	sensitiveKeyCache.set(k, result)

	return result
}

//nolint:cyclop,gocognit,gocyclo,nestif
func isSensitiveKeyASCIIFast(key []byte) (bool, bool) {
	start := -1
	prevIsLowerOrDigit := false
	prevIsUnderscore := true
	prevFirstOrLast := false

	for i := range key {
		c := key[i]

		if c >= 0x80 {
			return false, false
		}

		if isASCIIAlphaNum(c) {
			isUpper := c >= 'A' && c <= 'Z'
			if isUpper && prevIsLowerOrDigit && !prevIsUnderscore && start >= 0 {
				tok := key[start:i]
				if isSensitiveTokenASCII(tok) {
					return true, true
				}

				if prevFirstOrLast && equalsASCIIFold(tok, "name") {
					return true, true
				}

				prevFirstOrLast = equalsASCIIFold(tok, "first") || equalsASCIIFold(tok, "last")
				start = i
			} else if start < 0 {
				start = i
			}

			prevIsLowerOrDigit = (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')
			prevIsUnderscore = false

			continue
		}

		if start >= 0 {
			tok := key[start:i]
			if isSensitiveTokenASCII(tok) {
				return true, true
			}

			if prevFirstOrLast && equalsASCIIFold(tok, "name") {
				return true, true
			}

			prevFirstOrLast = equalsASCIIFold(tok, "first") || equalsASCIIFold(tok, "last")
			start = -1
		}

		prevIsLowerOrDigit = false
		prevIsUnderscore = true
	}

	if start >= 0 {
		tok := key[start:]
		if isSensitiveTokenASCII(tok) {
			return true, true
		}

		if prevFirstOrLast && equalsASCIIFold(tok, "name") {
			return true, true
		}
	}

	return false, true
}

func isASCIIAlphaNum(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

//nolint:cyclop,gocognit,gocyclo
func isSensitiveTokenASCII(tok []byte) bool {
	if len(tok) == 0 {
		return false
	}

	switch lowerASCIIByte(tok[0]) {
	case 'a':
		return equalsASCIIFold(tok, "acc") || equalsASCIIFold(tok, "account") || equalsASCIIFold(tok, "addr") ||
			equalsASCIIFold(tok, "address") || equalsASCIIFold(tok, "amount") || equalsASCIIFold(tok, "attestation") ||
			equalsASCIIFold(tok, "auth") || equalsASCIIFold(tok, "autograph")
	case 'b':
		return equalsASCIIFold(tok, "bal") || equalsASCIIFold(tok, "balance") || equalsASCIIFold(tok, "bearer") ||
			equalsASCIIFold(tok, "bill") || equalsASCIIFold(tok, "birth")
	case 'c':
		return equalsASCIIFold(tok, "card") || equalsASCIIFold(tok, "cc") || equalsASCIIFold(tok, "cell") ||
			equalsASCIIFold(tok, "cert") || equalsASCIIFold(tok, "checksum") || equalsASCIIFold(tok, "cookie") ||
			equalsASCIIFold(tok, "cred") || equalsASCIIFold(tok, "cv2") || equalsASCIIFold(tok, "cvc") ||
			equalsASCIIFold(tok, "cvv")
	case 'd':
		return equalsASCIIFold(tok, "dob") || equalsASCIIFold(tok, "dsa") || equalsASCIIFold(tok, "dsn")
	case 'e':
		return equalsASCIIFold(tok, "ecdsa") || equalsASCIIFold(tok, "email") || equalsASCIIFold(tok, "endorse") ||
			equalsASCIIFold(tok, "expiry")
	case 'f':
		return equalsASCIIFold(tok, "fingerprint")
	case 'h':
		return equalsASCIIFold(tok, "hash") || equalsASCIIFold(tok, "hmac") || equalsASCIIFold(tok, "holder")
	case 'i':
		return equalsASCIIFold(tok, "iban")
	case 'j':
		return equalsASCIIFold(tok, "jwt")
	case 'k':
		return equalsASCIIFold(tok, "key")
	case 'l':
		return equalsASCIIFold(tok, "login")
	case 'm':
		return equalsASCIIFold(tok, "mail")
	case 'p':
		return equalsASCIIFold(tok, "pass") || equalsASCIIFold(tok, "password") || equalsASCIIFold(tok, "pay") ||
			equalsASCIIFold(tok, "payment") || equalsASCIIFold(tok, "phone") || equalsASCIIFold(tok, "pkcs") ||
			equalsASCIIFold(tok, "pkcs12") || equalsASCIIFold(tok, "postal") || equalsASCIIFold(tok, "proof") ||
			equalsASCIIFold(tok, "pwd")
	case 'r':
		return equalsASCIIFold(tok, "rsa")
	case 's':
		return equalsASCIIFold(tok, "salt") || equalsASCIIFold(tok, "seal") || equalsASCIIFold(tok, "secret") ||
			equalsASCIIFold(tok, "secur") || equalsASCIIFold(tok, "secure") || equalsASCIIFold(tok, "security") ||
			equalsASCIIFold(tok, "sess") || equalsASCIIFold(tok, "session") || equalsASCIIFold(tok, "sgn") ||
			equalsASCIIFold(tok, "sid") || equalsASCIIFold(tok, "sig") || equalsASCIIFold(tok, "social") ||
			equalsASCIIFold(tok, "ssn") || equalsASCIIFold(tok, "swift")
	case 't':
		return equalsASCIIFold(tok, "tax") || equalsASCIIFold(tok, "tel") || equalsASCIIFold(tok, "telephone") ||
			equalsASCIIFold(tok, "token")
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
func isSensitiveNormalizedKey(normalized string) bool {
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
			if _, ok := sensitiveTokens[tok]; ok {
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
