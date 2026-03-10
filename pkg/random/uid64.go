package random

import (
	"strconv"
	"time"

	"github.com/tecnickcom/gogen/pkg/uhex"
)

// TUID64 holds the 64-bit random value.
type TUID64 uint64

// UID64 returns a 64-bit unique identifier where
// the upper 32 bits contain a timestamp and the lower 32 bits are random.
// The timestamp epoch is January 1st of the current decade's starting year.
func (r *Rnd) UID64() TUID64 {
	t := time.Now().UTC()
	offset := time.Date(t.Year()-t.Year()%10, 1, 1, 0, 0, 0, 0, time.UTC).Unix()

	return TUID64(uint64(((uint64)(t.Unix()-offset))<<32) + (uint64)(r.RandUint32())) //nolint:unconvert
}

// Hex returns the UID64 value as a fixed-length 16 digits hexadecimal string.
// This representation preserves the time-order.
func (u TUID64) Hex() string {
	return string(uhex.Hex64(uint64(u)))
}

// String returns the UID64 value as a base-36 variable-length string.
func (u TUID64) String() string {
	return strconv.FormatUint(uint64(u), 36)
}
