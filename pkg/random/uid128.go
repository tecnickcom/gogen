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

// UID128 generates a 128-bit unique identifier with high 64 bits from current time and low 64 bits random.
func (r *Rnd) UID128() TUID128 {
	return TUID128{
		t: (uint64)(time.Now().UnixNano()),
		r: r.RandUint64(),
	}
}

// Hex returns the UID128 as a 32-character hexadecimal string, preserving time-order.
func (u TUID128) Hex() string {
	return string(uhex.Hex64(u.t)) + string(uhex.Hex64(u.r))
}

// String returns the UID128 as a variable-length base-36 string (concatenated high and low parts).
func (u TUID128) String() string {
	return strconv.FormatUint(u.t, 36) + strconv.FormatUint(u.r, 36)
}
