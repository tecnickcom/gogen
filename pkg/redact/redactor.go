package redact

// Redactor is a configurable instance of the redaction engine (see
// [options.go] for the available configuration). The zero configuration
// (from [New] without options) behaves exactly like the package-level
// functions.
//
// A Redactor is immutable after construction and safe for concurrent use.
// Always construct one with [New]: the zero value ([Redactor]{}) is not the
// intended entry point, but it degrades to correct (uncached) redaction with
// the default marker rather than panicking — see usableRedactor.
type Redactor struct {
	marker      []byte
	disabled    Rule
	luhn        bool
	extraTokens map[string]struct{}
	dropTokens  map[string]struct{}
	keyMemo     *sensitiveKeyMemo
}

// defaultRedactor backs the package-level functions: standard marker, all
// rules enabled, built-in tokens, and the Luhn gate off.
var defaultRedactor = New() //nolint:gochecknoglobals

// Default returns the Redactor instance backing the package-level functions.
func Default() *Redactor {
	return defaultRedactor
}

// New builds a Redactor with the given options. Without options it matches
// the package-level behavior.
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
