package random

import (
	"strconv"
	"time"

	"github.com/tecnickcom/gogen/pkg/uhex"
)

// TUID128 holds the 128-bit random value.
type TUID128 struct {
	t uint64
	r uint64
}

// UID128 returns a 128-bit unique identifier.
// The high 64 bits are derived from the current time; the low 64 bits are random.
func (r *Rnd) UID128() TUID128 {
	return TUID128{
		t: (uint64)(time.Now().UnixNano()),
		r: r.RandUint64(),
	}
}

// Hex returns the UID128 value as a fixed-length 32 digits hexadecimal string.
// This representation preserves the time-order.
func (u TUID128) Hex() string {
	return string(uhex.Hex64(u.t)) + string(uhex.Hex64(u.r))
}

// String returns the UID128 value as a base-36 variable-length string.
func (u TUID128) String() string {
	return strconv.FormatUint(u.t, 36) + strconv.FormatUint(u.r, 36)
}
