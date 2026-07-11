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
	"errors"
	"fmt"
	"io"
	mrand "math/rand/v2"
	"strconv"

	"github.com/tecnickcom/nurago/pkg/uhex"
)

// ErrNegativeLength is returned when a negative length is requested from a
// size-taking helper such as [Rnd.RandomBytes] or [Rnd.RandString].
var ErrNegativeLength = errors.New("random: negative length")

// Character sets for random string generation.
const (
	chrDigits     = "0123456789"
	chrUppercase  = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	chrLowercase  = "abcdefghijklmnopqrstuvwxyz"
	chrSymbols    = "!#$%&()*+,-./:;<=>?@[]^_{|}~" // (exclude "\"'\\`")
	chrMapDefault = chrDigits + chrUppercase + chrLowercase + chrSymbols
	chrMapMaxLen  = 256
)

// Rnd defines the random number generator.
type Rnd struct {
	reader io.Reader
	chrMap []byte
}

// New constructs a random generator using the specified reader (or crypto/rand.Reader if nil) with optional customization.
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

// RandomBytes generates a slice of n random bytes, returning error if the reader fails.
//
// It uses [io.ReadFull] so the destination is always fully populated; a custom
// [io.Reader] that returns a short read with a nil error is treated as an error
// rather than silently truncating the randomness. A negative n returns
// [ErrNegativeLength] instead of panicking; n == 0 returns an empty slice.
func (r *Rnd) RandomBytes(n int) ([]byte, error) {
	if n < 0 {
		return nil, ErrNegativeLength
	}

	b := make([]byte, n)

	_, err := io.ReadFull(r.reader, b)
	if err != nil {
		return nil, fmt.Errorf("unable to generate %d random bytes: %w", n, err)
	}

	return b, nil
}

// RandUint32 returns a random 32-bit value, falling back to math/rand/v2 if crypto/rand fails.
func (r *Rnd) RandUint32() uint32 {
	b, err := r.RandomBytes(4)
	if err != nil {
		return mrand.Uint32() //nolint:gosec
	}

	return binary.LittleEndian.Uint32(b)
}

// RandUint64 returns a random 64-bit value, falling back to math/rand/v2 if crypto/rand fails.
func (r *Rnd) RandUint64() uint64 {
	b, err := r.RandomBytes(8)
	if err != nil {
		return mrand.Uint64() //nolint:gosec
	}

	return binary.LittleEndian.Uint64(b)
}

// RandHex64 returns a random 64-bit value as a 16-character hexadecimal string.
func (r *Rnd) RandHex64() string {
	return string(uhex.Hex64(r.RandUint64()))
}

// RandString64 returns a random 64-bit value as a variable-length base-36 string.
//
// It encodes a single value, so it is unambiguous, but it is variable-length;
// callers that need a fixed-width hexadecimal form should use [Rnd.RandHex64].
func (r *Rnd) RandString64() string {
	return strconv.FormatUint(r.RandUint64(), 36)
}

// RandString returns an n-character random string using the configured character map, suitable for passwords.
//
// Characters are selected with rejection sampling so each character map entry
// is drawn uniformly, even when 256 is not a multiple of the map length. A
// negative n returns [ErrNegativeLength] instead of panicking; n == 0 returns
// an empty string.
func (r *Rnd) RandString(n int) (string, error) {
	if n < 0 {
		return "", ErrNegativeLength
	}

	b := make([]byte, n)

	cmlen := len(r.chrMap)

	// limit is the largest multiple of cmlen not exceeding 256: random bytes
	// at or above it are rejected to avoid modulo bias.
	limit := 256 - (256 % cmlen)

	// Read entropy into the not-yet-finalized tail of b and map accepted bytes in
	// place towards the front. The write cursor (filled) never runs ahead of the
	// read cursor (j), so reusing b as the entropy scratch is safe: each byte is
	// read into a local before its slot may be overwritten. This keeps the whole
	// operation at one make plus the final string copy, regardless of how many
	// bytes rejection sampling discards, instead of allocating a fresh buffer on
	// every refill pass.
	filled := 0
	for filled < n {
		start := filled

		_, err := io.ReadFull(r.reader, b[start:])
		if err != nil {
			return "", fmt.Errorf("unable to generate %d random bytes: %w", n-start, err)
		}

		for j := start; j < n; j++ {
			v := b[j]
			if int(v) >= limit {
				continue
			}

			b[filled] = r.chrMap[int(v)%cmlen]
			filled++
		}
	}

	return string(b), nil
}
