/*
Package randkey provides utility functions to generate random uint64 keys in
different formats.

Deprecated: Please use github.com/tecnickcom/gogen/pkg/random instead.
*/
package randkey

import (
	"strconv"
	"strings"

	"github.com/tecnickcom/gogen/pkg/random"
)

var rnd = random.New(nil) //nolint:gochecknoglobals

// RandKey stores the random key.
//
// Deprecated: Please use random instead.
type RandKey struct {
	key uint64
}

// New generates a new uint64 random key.
//
// Deprecated: Please use random.RandUint64() instead.
func New() *RandKey {
	return &RandKey{key: rnd.RandUint64()}
}

// Key returns a uint64 key.
//
// Deprecated: Please use random.RandUint64() instead.
func (sk *RandKey) Key() uint64 {
	return sk.key
}

// String returns a variable-length string key.
//
// Deprecated: Please use random.RandString64() instead.
func (sk *RandKey) String() string {
	return strconv.FormatUint(sk.key, 36)
}

// Hex returns a fixed-length 16 digits hexadecimal string key.
//
// Deprecated: Please use random.RandHex64() instead.
func (sk *RandKey) Hex() string {
	s := strconv.FormatUint(sk.key, 16)

	slen := len(s)
	if slen < 16 {
		return strings.Repeat("0", (16-slen)) + s
	}

	return s
}
