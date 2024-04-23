package typeutil

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	mrand "math/rand/v2"
)

// RandReader is the default random number generator.
var RandReader = rand.Reader //nolint:gochecknoglobals

// RandomBytes generates a slice of random bytes with the specified length.
// The r argument must be a cryptographically secure random number generator (i.e. crypto/rand.Read).
func RandomBytes(r io.Reader, n int) ([]byte, error) {
	b := make([]byte, n)

	_, err := r.Read(b)
	if err != nil {
		return nil, fmt.Errorf("unable to generate %d random bytes: %w", n, err)
	}

	return b, nil
}

// RandUint32 returns a pseudo-random 32-bit value as a uint32 from the default Source.
// It try to use crypto/rand.Reader, if it fails, it falls back to math/rand/v2.Uint32.
func RandUint32() uint32 {
	b, err := RandomBytes(RandReader, 4)
	if err != nil {
		return mrand.Uint32()
	}

	return binary.LittleEndian.Uint32(b)
}

// RandUint64 returns a pseudo-random 64-bit value as a uint64 from the default Source.
// It try to use crypto/rand.Reader, if it fails, it falls back to math/rand/v2.Uint64.
func RandUint64() uint64 {
	b, err := RandomBytes(RandReader, 8)
	if err != nil {
		return mrand.Uint64()
	}

	return binary.LittleEndian.Uint64(b)
}
