/*
Package stringkey solves the practical problem of deriving a stable,
compact, non-cryptographic key from multiple text fields for fast lookup,
deduplication, and idempotency-style identifiers.

# Problem

Applications frequently need a deterministic key built from several strings
(for example: first name + last name + country, or provider + external ID +
scope). Building these keys manually is error-prone because equivalent inputs
may differ only by case, whitespace, or Unicode composition. stringkey applies
a canonical normalization pipeline before hashing so semantically equivalent
inputs map to the same key more often.

# How It Works

[New] builds a [StringKey] from one or more input fields using this pipeline:

 1. Trim leading and trailing Unicode whitespace for each field.
 2. Collapse repeated Unicode whitespace to a single space.
 3. Convert text to lowercase.
 4. Concatenate fields using a tab separator (`\t`) to preserve boundaries.
 5. Apply Unicode normalization (NFD -> NFC).
 6. Hash the resulting byte sequence with FarmHash64.

The resulting key is stored as an internal uint64 and can be retrieved in
multiple representations:

  - [StringKey.Key]: raw uint64 value.
  - [StringKey.String]: base-36 string (short, variable length).
  - [StringKey.Hex]: fixed 16-character lowercase hexadecimal string.

# Key Features

  - Deterministic canonicalization across case/spacing/Unicode variants.
  - Compact 64-bit key suitable for indexing and caching.
  - Multiple output formats for storage, logs, and URLs.
  - Fast, allocation-light hashing path for small field sets.

# Important Limits

This package is not cryptographically secure and must not be used for security
tokens, signatures, or untrusted collision-resistant identifiers.

It is designed for reasonably small input sizes and a moderate number of keys.
According to the birthday bound for a 64-bit hash space (~1.8x10^19):

  - collision probability is about 1% around 6.1x10^8 generated keys;
  - collision probability is about 50% around 5.1x10^9 generated keys.

Choose a larger hash or cryptographic construction when your scale or threat
model requires stronger collision guarantees.
*/
package stringkey

import (
	"bytes"
	"strconv"
	"strings"

	farmhash64 "github.com/tecnickcom/farmhash64/go/src"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// StringKey stores the encoded key.
type StringKey struct {
	key uint64
}

// New constructs deterministic FarmHash64 key from normalized input strings (whitespace/case/Unicode normalized).
func New(fields ...string) *StringKey {
	// NOTE: a fresh bytes.Buffer is allocated per call; for hot paths consider
	// a sync.Pool of buffers. The hash output must remain byte-stable because
	// it is used as a dedup/idempotency key, so this allocation is intentional.
	var b bytes.Buffer

	for _, v := range fields {
		// strings.Fields trims and collapses runs of Unicode whitespace
		// (unicode.IsSpace), matching strings.TrimSpace semantics.
		b.WriteString(strings.ToLower(strings.Join(strings.Fields(v), " ")))
		b.WriteByte('\t') // separate input strings
	}

	// norm.NFC alone is equivalent to chaining norm.NFD then norm.NFC:
	// NFC already decomposes (NFD) internally before recomposing, so the
	// resulting bytes (and therefore the hash) are identical.
	nb, _, _ := transform.Bytes(norm.NFC, b.Bytes())

	return &StringKey{key: farmhash64.FarmHash64(nb)}
}

// Key returns raw uint64 key value.
func (sk *StringKey) Key() uint64 {
	return sk.key
}

// String returns variable-length base-36 string representation of key.
func (sk *StringKey) String() string {
	return strconv.FormatUint(sk.key, 36)
}

// Hex returns fixed 16-character lowercase hexadecimal representation of key, zero-padded.
func (sk *StringKey) Hex() string {
	s := strconv.FormatUint(sk.key, 16)

	slen := len(s)
	if slen < 16 {
		return strings.Repeat("0", (16-slen)) + s
	}

	return s
}
