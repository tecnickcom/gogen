/*
Package random contains a collection of utility functions to generate random numbers and strings.
*/
package random

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	mrand "math/rand/v2"
)

const (
	chrDigits    = "0123456789"
	chrUppercase = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	chrLowercase = "abcdefghijklmnopqrstuvwxyz"
	chrSymbols   = "!#$%&()*+,-./:;<=>?@[]^_{|}~" // (exclude "\"'\\`")
	chrMap       = chrDigits + chrUppercase + chrLowercase + chrSymbols
	chrMapMaxLen = 256
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
		chrMap: []byte(chrMap),
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
		return mrand.Uint32()
	}

	return binary.LittleEndian.Uint32(b)
}

// RandUint64 returns a pseudo-random 64-bit value as a uint64 from the default Source.
// It try to use crypto/rand.Reader, if it fails, it falls back to math/rand/v2.Uint64.
func (r *Rnd) RandUint64() uint64 {
	b, err := r.RandomBytes(8)
	if err != nil {
		return mrand.Uint64()
	}

	return binary.LittleEndian.Uint64(b)
}

// RandString returns n-characters long random string that can be used as password.
// It generates n random bytes and maps them to characters using the default character set.
// The default character set can be overwritten by using the WithCharByteMap option.
func (r *Rnd) RandString(n int) (string, error) {
	b, err := r.RandomBytes(n)
	if err != nil {
		return "", err
	}

	for i, v := range b {
		b[i] = r.chrMap[(int(v)*len(chrMap))>>8]
	}

	return string(b), nil
}
