/*
Package randkey provides utility functions to generate random uint64 keys in different formats.
*/
package randkey

import (
	"strconv"
	"strings"

	"github.com/Vonage/gosrvlib/pkg/random"
)

// RandKey stores the random key.
type RandKey struct {
	key uint64
}

// New generates a new uint64 random key.
func New() *RandKey {
	return &RandKey{key: random.New(nil).RandUint64()}
}

// Key returns a uint64 key.
func (sk *RandKey) Key() uint64 {
	return sk.key
}

// String returns a variable-length string key.
func (sk *RandKey) String() string {
	return strconv.FormatUint(sk.key, 36)
}

// Hex returns a fixed-length 16 digits hexadecimal string key.
func (sk *RandKey) Hex() string {
	s := strconv.FormatUint(sk.key, 16)

	slen := len(s)
	if slen < 16 {
		return strings.Repeat("0", (16-slen)) + s
	}

	return s
}
