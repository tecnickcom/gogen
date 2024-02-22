/*
Package uid provides functions to generate simple time-and-random-based unique identifiers.

It provides functions to generate 64 and 128 bit random identifiers in base-36 string format.

This package's outputs might be easily predictable.
For random IDs suitable for security-sensitive work, please the uidc package instead.
*/
package uid

//nolint:gci
import (
	"math/rand/v2"
	"strconv"
	"time"
)

// NewID64 generates and returns a new base-36-string-formatted 64 bit unique ID based on time (high 32 bit) and a random number (low 32 bit).
// NOTE: the zero time is set to the 1st of January of 10 years ago.
func NewID64() string {
	t := time.Now().UTC()
	offset := time.Date(t.Year()-10, 1, 1, 0, 0, 0, 0, time.UTC).Unix() // [s] time starts 1st JAN 10 years ago

	return strconv.FormatUint((((uint64)(t.Unix()-offset))<<32)+(uint64)(rand.Uint32()), 36) // #nosec
}

// NewID128 generates and returns a new base-36-string-formatted 128 bit unique ID based on time (high 64 bit) and a random number (low 64 bit).
func NewID128() string {
	return strconv.FormatUint((uint64)(time.Now().UTC().UnixNano()), 36) + strconv.FormatUint(rand.Uint64(), 36) // #nosec
}
