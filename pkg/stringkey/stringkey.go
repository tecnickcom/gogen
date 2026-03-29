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

 1. Trim leading and trailing spaces for each field.
 2. Collapse repeated whitespace to a single space.
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
	"regexp"
	"strconv"
	"strings"

	farmhash64 "github.com/tecnickcom/farmhash64/go/src"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// regexPatternEmptySpaces matches one or more consecutive whitespace characters.
const regexPatternEmptySpaces = `\s{1,}`

// regexEmptySpaces is the compiled regular expression for matching multiple spaces.
var regexEmptySpaces = regexp.MustCompile(regexPatternEmptySpaces)

// StringKey stores the encoded key.
type StringKey struct {
	key uint64
}

// New encode (hash) input strings into a uint64 key.
func New(fields ...string) *StringKey {
	var b bytes.Buffer

	for _, v := range fields {
		b.WriteString(strings.ToLower(regexEmptySpaces.ReplaceAllLiteralString(strings.TrimSpace(v), " ")))
		b.WriteByte('\t') // separate input strings
	}

	nb, _, _ := transform.Bytes(transform.Chain(norm.NFD, norm.NFC), b.Bytes())

	return &StringKey{key: farmhash64.FarmHash64(nb)}
}

// Key returns a uint64 key.
func (sk *StringKey) Key() uint64 {
	return sk.key
}

// String returns a variable-length string key.
func (sk *StringKey) String() string {
	return strconv.FormatUint(sk.key, 36)
}

// Hex returns a fixed-length 16 digits hexadecimal string key.
func (sk *StringKey) Hex() string {
	s := strconv.FormatUint(sk.key, 16)

	slen := len(s)
	if slen < 16 {
		return strings.Repeat("0", (16-slen)) + s
	}

	return s
}
