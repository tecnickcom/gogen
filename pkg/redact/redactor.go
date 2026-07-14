package redact

import "unsafe"

// Redactor is a configurable instance of the redaction engine (see
// [options.go] for the available configuration) and the only entry point to it.
//
// A Redactor is immutable after construction and safe for concurrent use.
// Always obtain one with [Default] (the shared, zero-configuration instance) or
// [New] (an independent, configured one): the zero value ([Redactor]{}) is not
// an intended entry point, but it degrades to correct (uncached) redaction with
// the default marker rather than panicking — see usableRedactor.
type Redactor struct {
	marker      []byte
	disabled    Rule
	luhn        bool
	extraTokens map[string]struct{}
	dropTokens  map[string]struct{}
	keyMemo     *sensitiveKeyMemo
}

// defaultRedactor is the shared zero-configuration instance returned by
// [Default]: standard marker, all rules enabled, built-in tokens, and the Luhn
// gate off.
var defaultRedactor = New() //nolint:gochecknoglobals

// Default returns the shared, zero-configuration Redactor: the standard marker,
// all rules enabled, the built-in token set, and the Luhn gate off. It is the
// redactor the httpclient, httpserver, and httpreverseproxy packages fall back
// to when no redact function is configured.
//
// Prefer it over New() for the default configuration: a Redactor memoizes
// non-ASCII key classifications per instance, so sharing one keeps that cache
// shared and bounded instead of allocating a fresh one per caller.
func Default() *Redactor {
	return defaultRedactor
}

// New builds an independent Redactor with the given options. Without options it
// is configured like [Default] but carries its own key-classification cache;
// callers that want the default configuration should use [Default] instead.
func New(opts ...Option) *Redactor {
	re := &Redactor{
		marker:  redactedBytes,
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
	// AppendTo is the only entry point whose destination is caller-controlled
	// and could alias src (an in-place AppendTo(b, b)); redacting into an
	// aliasing buffer would let the write cursor overtake the read cursor and
	// corrupt — and leak — not-yet-scanned bytes. Bytes and Pooled always pass a
	// fresh or pooled destination, so they skip this check.
	if backingOverlap(dst, src) {
		dst = make([]byte, 0, len(src))
	}

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

// usableRedactor returns a Redactor safe to run: for a [New]-built instance it
// is the receiver unchanged, and for the zero value it is a defensive copy with
// the nil marker and key memo filled in. The copy is not mutated in place, so a
// zero value shared across goroutines cannot race here; it is uncached (a fresh
// memo per call), which is acceptable because the zero value is not the intended
// construction path (see [New]).
func (re *Redactor) usableRedactor() *Redactor {
	if re.marker != nil && re.keyMemo != nil {
		return re
	}

	clone := *re
	if clone.marker == nil {
		clone.marker = redactedBytes
	}

	if clone.keyMemo == nil {
		clone.keyMemo = newSensitiveKeyMemo()
	}

	return &clone
}

// enabled reports whether the given rule class is active on this instance.
func (re *Redactor) enabled(r Rule) bool {
	return re.disabled&r == 0
}

// backingOverlap reports whether the backing arrays of a and b overlap in
// memory, using the same address-range test the standard library's crypto
// packages use to reject in-place aliasing. Only [Redactor.AppendTo] needs it:
// it is the sole entry point whose destination is caller-controlled.
func backingOverlap(a, b []byte) bool {
	if cap(a) == 0 || cap(b) == 0 {
		return false
	}

	aBeg := uintptr(unsafe.Pointer(unsafe.SliceData(a)))
	bBeg := uintptr(unsafe.Pointer(unsafe.SliceData(b)))

	return aBeg < bBeg+uintptr(cap(b)) && bBeg < aBeg+uintptr(cap(a))
}
