package random

import (
	"strconv"
	"time"
)

// UID64 returns a 64-bit unique ID encoded as a base-36 string.
// The upper 32 bits contain a timestamp and the lower 32 bits are random.
// The timestamp epoch is January 1st of the current decade's starting year.
func (r *Rnd) UID64() string {
	t := time.Now().UTC()
	offset := time.Date(t.Year()-t.Year()%10, 1, 1, 0, 0, 0, 0, time.UTC).Unix()

	return strconv.FormatUint((((uint64)(t.Unix()-offset))<<32)+(uint64)(r.RandUint32()), 36)
}

// UID128 returns a 128-bit unique identifier encoded as a base-36 string.
// The high 64 bits are derived from the current time; the low 64 bits are random.
func (r *Rnd) UID128() string {
	return strconv.FormatUint((uint64)(time.Now().UnixNano()), 36) + strconv.FormatUint(r.RandUint64(), 36)
}
