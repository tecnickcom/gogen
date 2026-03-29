package random

import (
	"strconv"
	"time"

	"github.com/tecnickcom/gogen/pkg/uhex"
)

// TUID64 holds the 64-bit random value.
type TUID64 uint64

// UID64 generates a 64-bit unique identifier with upper 32 bits as a time-decade offset and lower 32 bits random.
func (r *Rnd) UID64() TUID64 {
	t := time.Now().UTC()
	offset := time.Date(t.Year()-t.Year()%10, 1, 1, 0, 0, 0, 0, time.UTC).Unix()

	return TUID64(uint64(((uint64)(t.Unix()-offset))<<32) + (uint64)(r.RandUint32())) //nolint:unconvert
}

// Hex returns the UID64 as a 16-character hexadecimal string, preserving time-order.
func (u TUID64) Hex() string {
	return string(uhex.Hex64(uint64(u)))
}

// String returns the UID64 as a variable-length base-36 string.
func (u TUID64) String() string {
	return strconv.FormatUint(uint64(u), 36)
}
