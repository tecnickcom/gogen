package redact

import (
	"sync/atomic"
)

// Redactor is a configurable instance of the redaction engine (see
// [options.go] for the available configuration). The zero configuration
// (from [New] without options) behaves exactly like the package-level
// functions, except that its Luhn gate is instance-scoped and starts
// disabled instead of following [SetLuhnCheck].
//
// A Redactor is immutable after construction and safe for concurrent use.
type Redactor struct {
	marker      []byte
	disabled    Rule
	luhn        *atomic.Bool
	extraTokens map[string]struct{}
	dropTokens  map[string]struct{}
	keyMemo     *sensitiveKeyMemo
}

// defaultRedactor backs the package-level functions: standard marker, all
// rules enabled, built-in tokens, and the process-wide Luhn toggle.
var defaultRedactor = &Redactor{ //nolint:gochecknoglobals
	marker:  redactedBytes,
	luhn:    &luhnCheckEnabled,
	keyMemo: sensitiveKeyCache,
}

// Default returns the Redactor instance backing the package-level functions.
func Default() *Redactor {
	return defaultRedactor
}

// New builds a Redactor with the given options. Without options it matches
// the package-level behavior (with an instance-scoped Luhn gate, default
// off).
func New(opts ...Option) *Redactor {
	re := &Redactor{
		marker:  redactedBytes,
		luhn:    new(atomic.Bool),
		keyMemo: newSensitiveKeyMemo(),
	}

	for _, opt := range opts {
		if opt != nil {
			opt(re)
		}
	}

	return re
}

// String redacts sensitive data from s with this instance's configuration
// and returns the sanitized string.
func (re *Redactor) String(s string) string {
	return re.BytesToString([]byte(s))
}

// Bytes redacts sensitive data from b and returns the result as a new byte
// slice. The input is never modified.
func (re *Redactor) Bytes(b []byte) []byte {
	return re.redactInto(make([]byte, 0, len(b)), b)
}

// AppendTo redacts sensitive data from src and appends the result into dst
// (after resetting its length to zero); see the package-level [AppendTo].
func (re *Redactor) AppendTo(dst, src []byte) []byte {
	return re.redactInto(dst, src)
}

// Pooled redacts sensitive data from src using the shared pooled buffer and
// passes the result to consume; see the package-level [Pooled].
func (re *Redactor) Pooled(src []byte, consume func([]byte)) {
	if consume == nil {
		return
	}

	dst := getPooledRedactionBuffer(len(src))
	out := re.redactInto(dst, src)

	// Deferred so the buffer is returned to the pool even if consume panics.
	defer putPooledRedactionBuffer(out)

	consume(out)
}

// BytesToString redacts sensitive data from a byte slice and returns the
// result as a string; see the package-level [BytesToString].
func (re *Redactor) BytesToString(b []byte) string {
	var out string

	re.Pooled(b, func(redacted []byte) {
		out = string(redacted)
	})

	return out
}

// enabled reports whether the given rule class is active on this instance.
func (re *Redactor) enabled(r Rule) bool {
	return re.disabled&r == 0
}

// markerEndsAt reports whether this instance's marker occupies the bytes
// ending at src[j] (inclusive).
func (re *Redactor) markerEndsAt(src []byte, j int) bool {
	start := j + 1 - len(re.marker)
	if start < 0 || src[j] != re.marker[len(re.marker)-1] {
		return false
	}

	for k, c := range re.marker {
		if src[start+k] != c {
			return false
		}
	}

	return true
}

// markerAt reports whether this instance's marker starts at src[i].
func (re *Redactor) markerAt(src []byte, i int) bool {
	if i+len(re.marker) > len(src) {
		return false
	}

	for k, c := range re.marker {
		if src[i+k] != c {
			return false
		}
	}

	return true
}
