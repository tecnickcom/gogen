/*
Package uidc provides functions to generate simple time-and-random-based unique
identifiers.

It offers functions to generate 64-bit and 128-bit random identifiers in base-36
string format.

The generated IDs are not intended for cryptographic or security-related
purposes. Instead, they serve as simple unique identifiers for various use
cases.
*/
package uidc

import (
	"strconv"
	"time"

	"github.com/tecnickcom/gogen/pkg/random"
)

// NewID64 returns a 64-bit unique ID encoded as a base-36 string.
// The upper 32 bits contain a timestamp and the lower 32 bits are random.
// The timestamp epoch is January 1st of the current decade's starting year.
func NewID64() string {
	t := time.Now().UTC()
	offset := time.Date(t.Year()-t.Year()%10, 1, 1, 0, 0, 0, 0, time.UTC).Unix()

	return strconv.FormatUint((((uint64)(t.Unix()-offset))<<32)+(uint64)(random.New(nil).RandUint32()), 36)
}

// NewID128 returns a 128-bit unique identifier encoded as a base-36 string.
// The high 64 bits are derived from the current time; the low 64 bits are random.
func NewID128() string {
	return strconv.FormatUint((uint64)(time.Now().UTC().UnixNano()), 36) + strconv.FormatUint(random.New(nil).RandUint64(), 36)
}
