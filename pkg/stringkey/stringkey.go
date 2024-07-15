/*
Package stringkey provides the ability to create a simple unique hash key from
multiple strings.

This package uses the farmhash64 algorithm to create a 64-bit hash from the
concatenation of input strings. The input strings are tab-concatenated,
unicode-normalized, converted to lowercase, and stripped of leading, trailing,
and multiple spaces.

This package is designed for working with a small number of strings and is not
intended for cryptographic purposes. The total number of input bytes should be
reasonably small to be compatible with a 64-bit hash.

According to the birthday problem, with a 64-bit hash, the hash space is
1.8x10^19 and the probability of a collision is 1% with 6.1×10^8 keys and 50%
with 5.1×10^9 keys.
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

const regexPatternEmptySpaces = `\s{1,}`

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
