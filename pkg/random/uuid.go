package random

import (
	"encoding/hex"
	"time"
)

// UUID is the type for 128 bit (16 byte) Universally Unique IDentifiers (UUIDs)
// as defined in RFC 9562 (https://datatracker.ietf.org/doc/html/rfc9562).
type UUID [16]byte

const (
	// uuidSep is the character used as separator between UUID sections.
	uuidSep = '-'
)

// UUIDv7 returns 128-bit Universally Unique IDentifier (UUID) version 7 as defined by
// RFC 9562 (https://datatracker.ietf.org/doc/html/rfc9562#name-uuid-version-7).
func (r *Rnd) UUIDv7() UUID {
	var ub UUID

	now := time.Now()
	ms := now.UnixMilli()

	// unix_ts_ms:
	// 48-bit big-endian unsigned number of the Unix Epoch timestamp in milliseconds
	// bits 0 through 47 (octets 0-5)
	ub[0] = byte(ms >> 40)
	ub[1] = byte(ms >> 32)
	ub[2] = byte(ms >> 24)
	ub[3] = byte(ms >> 16)
	ub[4] = byte(ms >> 8)
	ub[5] = byte(ms)

	// approximate map remaining nanoseconds in the range [0, 4095]
	// Note:
	// 4294 = (4095 * 2^20) / 1,000,000
	// shifting right by 20 bits is roughly equivalent to dividing byiding by 1,048,576.
	ns := ((now.Nanosecond() % 1e6) * 4294) >> 20

	// ver:
	// 4-bit version field set to 0b0111 (7)
	// bits 48 through 51 of octet 6
	// +
	// rand_a:
	// 12 bits of pseudorandom data
	// bits 52 through 63 (octets 6-7)
	ub[6] = 0x70 | (0x0F & byte(ns>>8))
	ub[7] = byte(ns)

	// generate 8 random bytes
	rb, _ := r.RandomBytes(8)

	// var:
	// 2-bit variant field set to 0b10
	// bits 64 and 65 of octet 8.
	// +
	// rand_b:
	// 62 bits of pseudorandom data to provide uniqueness
	// bits 66 through 127 (octets 8-15).
	ub[8] = 0x80 | (0x3F & rb[0])
	copy(ub[9:16], rb[1:8])

	return ub
}

// Byte returns the UUID byte-slice representation:
// []byte("xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx").
func (u UUID) Byte() []byte {
	uuid := make([]byte, 36)

	hex.Encode(uuid[0:8], u[0:4])
	uuid[8] = uuidSep
	hex.Encode(uuid[9:13], u[4:6])
	uuid[13] = uuidSep
	hex.Encode(uuid[14:18], u[6:8])
	uuid[18] = uuidSep
	hex.Encode(uuid[19:23], u[8:10])
	uuid[23] = uuidSep
	hex.Encode(uuid[24:36], u[10:16])

	return uuid
}

// String returns the UUID string representation:
// "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx".
func (u UUID) String() string {
	return string(u.Byte())
}
