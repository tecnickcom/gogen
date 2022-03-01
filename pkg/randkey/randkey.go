// Package randkey creates a random uint64 key.
package randkey

import (
	"crypto/rand"
	"encoding/binary"
	"strconv"
	"strings"
)

// RandKey stores the random key.
type RandKey struct {
	key uint64
}

// New generates a new uint64 random key.
func New() *RandKey {
	b := make([]byte, 8) // 8 bytes for 64 bit
	_, _ = rand.Read(b)

	return &RandKey{key: binary.LittleEndian.Uint64(b)}
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
