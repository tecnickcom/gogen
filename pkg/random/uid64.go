package random

import (
	"strconv"
	"time"

	"github.com/tecnickcom/nurago/pkg/uhex"
)

// TUID64 is a time-ordered 64-bit identifier: the upper 32 bits are a
// decade-relative second offset and the lower 32 bits are random.
type TUID64 uint64

// UID64 generates a 64-bit unique identifier with upper 32 bits as a time-decade offset and lower 32 bits random.
//
// Because only the lower 32 bits are random and the upper 32 bits are shared by
// all identifiers generated within the same second, uniqueness relies on a
// single 32-bit random draw per second: birthday collisions become likely at
// roughly 2^16 (~65k) identifiers generated within the same second, about a 39%
// chance at 65k, and 50% at ~77k. For high-throughput or collision-critical use,
// prefer [Rnd.UID128] (64 random bits) or [Rnd.UUIDv7] (62 random bits; its other
// 12 non-timestamp bits are clock-derived, not random).
//
// The second offset is measured from the start of the current decade, so it
// stays well within 32 bits; it resets at each decade boundary, which makes the
// value time-ordered only within a decade (see [TUID64.Hex]).
func (r *Rnd) UID64() TUID64 {
	t := time.Now().UTC()

	// Offset from the start of the current decade (Jan 1, 00:00 UTC). Compute the
	// year once rather than evaluating t.Year() twice.
	year := t.Year()
	offset := time.Date(year-year%10, 1, 1, 0, 0, 0, 0, time.UTC).Unix()

	return TUID64((uint64(t.Unix()-offset) << 32) + uint64(r.RandUint32()))
}

// Format writes the UID64 as its 16-character hexadecimal form into dst.
//
// Format writes directly into the caller-provided array and allocates nothing,
// so prefer it in performance-critical code. For a string use [TUID64.Hex]; for
// a byte slice use [TUID64.Byte]. Values are time-ordered within a decade (see
// [Rnd.UID64]).
func (u TUID64) Format(dst *[16]byte) {
	uhex.Hex64UB(uint64(u), dst)
}

// Byte returns the UID64 as its 16-character hexadecimal form in a byte slice.
//
// Byte fills a local [16]byte array and returns a slice over it, so whether it
// allocates depends on escape analysis at the call site: if the returned slice
// does not escape the caller (it is read and discarded, or only its bytes are
// consumed), the array stays on the stack and no allocation occurs; if it escapes
// (stored, returned, sent on a channel, or passed to a function the compiler
// cannot inline), the array is heap-allocated. Use [TUID64.Hex] for a string, or
// [TUID64.Format] to write into a pre-allocated buffer and never allocate
// regardless of escape analysis.
func (u TUID64) Byte() []byte {
	var b [16]byte

	u.Format(&b)

	return b[:]
}

// Hex returns the UID64 as a 16-character hexadecimal string.
//
// Values are time-ordered within a decade; ordering is discontinuous across
// decade boundaries because the underlying second offset resets (see [Rnd.UID64]).
func (u TUID64) Hex() string {
	return string(u.Byte())
}

// String returns the UID64 as a variable-length base-36 string.
//
// Unlike [TUID128.String], this encodes a single value and is therefore
// unambiguous and round-trippable. It is, however, variable-length; callers
// that need a fixed-width representation should use [TUID64.Hex].
func (u TUID64) String() string {
	return strconv.FormatUint(uint64(u), 36)
}
