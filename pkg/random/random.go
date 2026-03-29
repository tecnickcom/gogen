/*
Package random provides utility functions for generating random bytes, numeric
identifiers, UID/UUID values, hexadecimal/base36 IDs, and configurable random
strings.

# Problem

Application code frequently needs random values for IDs, tokens, test fixtures,
or temporary credentials. Repeatedly wiring secure randomness, string encoding,
and character-set mapping at call sites is error-prone and inconsistent.

This package centralizes those patterns in one small API.

# Randomness Source

By default, [New] uses [crypto/rand.Reader], which is suitable for
security-sensitive randomness. A custom [io.Reader] can be supplied directly to
[New] for testing or specialized environments.

For [Rnd.RandUint32] and [Rnd.RandUint64], if secure random byte generation
fails, the implementation falls back to math/rand/v2 to keep call sites
non-failing. This makes those helpers resilient but less suitable for strict
cryptographic guarantees when fallback occurs.

# What It Provides

  - [Rnd.RandomBytes] for raw random byte slices.
  - [Rnd.RandUint32] and [Rnd.RandUint64] for random integers.
  - [Rnd.RandHex64] for fixed-length 16-char hexadecimal IDs.
  - [Rnd.RandString64] for compact base-36 IDs.
  - [Rnd.RandString] for random strings of length n using a configurable
    byte-to-character map.
  - [Rnd.UID64] for time-aware 64-bit unique identifiers ([TUID64]) with
    [TUID64.Hex] and [TUID64.String] formats.
  - [Rnd.UID128] for time+random 128-bit unique identifiers ([TUID128]) with
    [TUID128.Hex] and [TUID128.String] formats.
  - [Rnd.UUIDv7] for RFC 9562 UUID version 7 values ([UUID]).

# Character Map Customization

[Rnd.RandString] uses a default map containing digits, uppercase/lowercase
letters, and symbols. You can override it with [WithByteToCharMap].

  - Empty map input restores the default map.
  - Maps longer than 256 bytes are truncated to 256.

# Usage

	r := random.New(nil) // default: crypto/rand.Reader

	id := r.RandHex64()
	short := r.RandString64()
	_ = id
	_ = short

	pwd, err := r.RandString(24)
	if err != nil {
	    return err
	}
	_ = pwd

	uid64 := r.UID64()
	uid128 := r.UID128()
	uuid := r.UUIDv7()
	_ = uid64.Hex()
	_ = uid128.String()
	_ = uuid.String()

	alphaNum := []byte("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	r2 := random.New(nil, random.WithByteToCharMap(alphaNum))
	_, _ = r2.RandString(16)

This package is ideal for Go services that need convenient, reusable random
value generation with sensible secure defaults and explicit customization.
*/
package random

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	mrand "math/rand/v2"
	"strconv"

	"github.com/tecnickcom/gogen/pkg/uhex"
)

// Character sets for random string generation.
const (
	chrDigits     = "0123456789"
	chrUppercase  = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	chrLowercase  = "abcdefghijklmnopqrstuvwxyz"
	chrSymbols    = "!#$%&()*+,-./:;<=>?@[]^_{|}~" // (exclude "\"'\\`")
	chrMapDefault = chrDigits + chrUppercase + chrLowercase + chrSymbols
	chrMapMaxLen  = 256
)

// Rnd defines then random number generator.
type Rnd struct {
	reader io.Reader
	chrMap []byte
}

// New initialize the random reader.
// The r argument must be a cryptographically secure random number generator.
// The crypto/rand.Read is used as default if r == nil.
func New(r io.Reader, opts ...Option) *Rnd {
	if r == nil {
		r = rand.Reader
	}

	rnd := &Rnd{
		reader: r,
		chrMap: []byte(chrMapDefault),
	}

	for _, applyOpt := range opts {
		applyOpt(rnd)
	}

	return rnd
}

// RandomBytes generates a slice of random bytes with the specified length.
func (r *Rnd) RandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)

	_, err := r.reader.Read(b)
	if err != nil {
		return nil, fmt.Errorf("unable to generate %d random bytes: %w", n, err)
	}

	return b, nil
}

// RandUint32 returns a pseudo-random 32-bit value as a uint32 from the default Source.
// It try to use crypto/rand.Reader, if it fails, it falls back to math/rand/v2.Uint32.
func (r *Rnd) RandUint32() uint32 {
	b, err := r.RandomBytes(4)
	if err != nil {
		return mrand.Uint32() //nolint:gosec
	}

	return binary.LittleEndian.Uint32(b)
}

// RandUint64 returns a pseudo-random 64-bit value as a uint64 from the default Source.
// It try to use crypto/rand.Reader, if it fails, it falls back to math/rand/v2.Uint64.
func (r *Rnd) RandUint64() uint64 {
	b, err := r.RandomBytes(8)
	if err != nil {
		return mrand.Uint64() //nolint:gosec
	}

	return binary.LittleEndian.Uint64(b)
}

// RandHex64 returns a pseudo-random 64-bit value as a fixed-length 16 digits hexadecimal string.
func (r *Rnd) RandHex64() string {
	return string(uhex.Hex64(r.RandUint64()))
}

// RandString64 returns a pseudo-random 64-bit value as a base-36 variable-length string.
func (r *Rnd) RandString64() string {
	return strconv.FormatUint(r.RandUint64(), 36)
}

// RandString returns n-characters long random string that can be used as password.
// It generates n random bytes and maps them to characters using the default character set.
// The default character set can be overwritten by using the WithCharByteMap option.
func (r *Rnd) RandString(n int) (string, error) {
	b, err := r.RandomBytes(n)
	if err != nil {
		return "", err
	}

	cmlen := len(r.chrMap)

	for i, v := range b {
		b[i] = r.chrMap[(int(v)*cmlen)>>8]
	}

	return string(b), nil
}
