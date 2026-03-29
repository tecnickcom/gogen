/*
Package uhex solves the performance problem of converting unsigned integers to
their hexadecimal string representation in hot paths — trace ID generation, log
formatting, binary protocol encoding — where the allocations and reflection
overhead of [fmt.Sprintf] or [encoding/hex] are unacceptable.

# Problem

The standard library routes hex encoding through [fmt.Sprintf] ("%x"), which
uses reflection and allocates, or through [encoding/hex], which operates on
byte slices but still requires a pre-allocated destination. Neither is ideal
when encoding millions of IDs per second with zero heap pressure. uhex trades
generality for speed: one stack-allocated fixed-size array per integer width,
filled by direct nibble-index lookups into a 16-byte constant table — no
reflection, no heap allocation in the hot path.

# How It Works

Each function allocates a fixed-size array on the stack, fills every byte by
shifting the input right and masking the low four bits to index into the
constant lookup table "0123456789abcdef", then returns a slice of that array.
No branches, no loops, no allocations beyond the returned slice header.

  - [Hex64] encodes a uint64 to a 16-byte lowercase hex slice.
  - [Hex32] encodes a uint32 to an 8-byte lowercase hex slice.
  - [Hex16] encodes a uint16 to a 4-byte lowercase hex slice.
  - [Hex8]  encodes a uint8  to a 2-byte lowercase hex slice.

Output is always zero-padded to the full width of the integer type, making it
safe to concatenate or compare without length checks.

# Usage

	id := uhex.Hex64(traceID)   // e.g. []byte("0000abcd1234ef56")
	b := uhex.Hex32(checksum)   // e.g. []byte("deadbeef")
	s := string(uhex.Hex8(tag)) // e.g. "ff"

This package is ideal for any Go application that needs high-throughput,
allocation-sensitive hex encoding of fixed-width unsigned integers.
*/
package uhex

const hexTable = "0123456789abcdef"

// Hex64 converts uint64 to 16-byte lowercase hexadecimal slice, zero-padded and allocated on stack.
func Hex64(n uint64) []byte {
	var buf [16]byte

	buf[0] = hexTable[n>>60&0xf]
	buf[1] = hexTable[n>>56&0xf]
	buf[2] = hexTable[n>>52&0xf]
	buf[3] = hexTable[n>>48&0xf]
	buf[4] = hexTable[n>>44&0xf]
	buf[5] = hexTable[n>>40&0xf]
	buf[6] = hexTable[n>>36&0xf]
	buf[7] = hexTable[n>>32&0xf]
	buf[8] = hexTable[n>>28&0xf]
	buf[9] = hexTable[n>>24&0xf]
	buf[10] = hexTable[n>>20&0xf]
	buf[11] = hexTable[n>>16&0xf]
	buf[12] = hexTable[n>>12&0xf]
	buf[13] = hexTable[n>>8&0xf]
	buf[14] = hexTable[n>>4&0xf]
	buf[15] = hexTable[n&0xf]

	return buf[:]
}

// Hex32 converts uint32 to 8-byte lowercase hexadecimal slice, zero-padded and allocated on stack.
func Hex32(n uint32) []byte {
	var buf [8]byte

	buf[0] = hexTable[n>>28&0xf]
	buf[1] = hexTable[n>>24&0xf]
	buf[2] = hexTable[n>>20&0xf]
	buf[3] = hexTable[n>>16&0xf]
	buf[4] = hexTable[n>>12&0xf]
	buf[5] = hexTable[n>>8&0xf]
	buf[6] = hexTable[n>>4&0xf]
	buf[7] = hexTable[n&0xf]

	return buf[:]
}

// Hex16 converts uint16 to 4-byte lowercase hexadecimal slice, zero-padded and allocated on stack.
func Hex16(n uint16) []byte {
	var buf [4]byte

	buf[0] = hexTable[n>>12&0xf]
	buf[1] = hexTable[n>>8&0xf]
	buf[2] = hexTable[n>>4&0xf]
	buf[3] = hexTable[n&0xf]

	return buf[:]
}

// Hex8 converts uint8 to 2-byte lowercase hexadecimal slice, zero-padded and allocated on stack.
func Hex8(n uint8) []byte {
	var buf [2]byte

	buf[0] = hexTable[n>>4&0xf]
	buf[1] = hexTable[n&0xf]

	return buf[:]
}
