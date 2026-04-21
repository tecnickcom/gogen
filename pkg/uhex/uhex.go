/*
Package uhex provides fixed-width, lowercase hexadecimal encoders for unsigned
integers and fixed-size byte arrays.

It is intended for hot paths such as trace ID generation, log formatting, and
binary protocol work where general-purpose formatting can be unnecessarily
expensive. The implementation is deliberately simple: each output byte is
produced by shifting or masking a nibble and indexing into the constant table
"0123456789abcdef".

Compared with generic formatting such as [fmt.Sprintf]("%x", v), uhex avoids
reflection and width handling. Compared with [encoding/hex], it focuses on
fixed-width values and exposes helpers that write directly into caller-owned
arrays.

# Output

All encoders produce lowercase hexadecimal.

Integer-based helpers always zero-pad to the full width of the input type:

  - [Hex64] and [Hex64UB] produce 16 bytes.
  - [Hex32] and [Hex32UB] produce 8 bytes.
  - [Hex16] and [Hex16UB] produce 4 bytes.
  - [Hex8] and [Hex8UB] produce 2 bytes.

Byte-array helpers encode each input byte in order, two hex characters per
byte:

  - [Hex64B] and [Hex64BB] encode [8]byte values.
  - [Hex32B] and [Hex32BB] encode [4]byte values.
  - [Hex16B] and [Hex16BB] encode [2]byte values.
  - [Hex8B] and [Hex8BB] encode [1]byte values.

# Allocation Behavior

The slice-returning helpers ([Hex64], [Hex32], [Hex16], [Hex8], [Hex64B],
[Hex32B], [Hex16B], and [Hex8B]) are the most convenient API, but the returned
slice may require an allocation if it escapes.

For allocation-sensitive code, prefer the buffer-writing helpers that accept a
destination array pointer. [Hex64UB], [Hex32UB], [Hex16UB], and [Hex8UB] write
integer encodings into caller-owned buffers. [Hex64BB], [Hex32BB], [Hex16BB],
and [Hex8BB] do the same for fixed-size byte arrays.

# Naming

Function suffixes follow this pattern:

  - No suffix: encode an unsigned integer and return a []byte.
  - UB: encode an unsigned integer into a caller-provided buffer.
  - B: encode a fixed-size byte array and return a []byte.
  - BB: encode a fixed-size byte array into a caller-provided buffer.

# Usage

	id := uhex.Hex64(traceID)
	sum := uhex.Hex32(checksum)
	tag := string(uhex.Hex8(kind))

	var dst [16]byte
	uhex.Hex64UB(traceID, &dst)
	out := dst[:]

	src := [8]byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef}
	uhex.Hex64BB(src, &dst)

uhex is a good fit when the input width is known in advance and predictable,
lowercase, zero-padded hexadecimal output is required.
*/
package uhex

const hexTable = "0123456789abcdef"

// Hex64 returns the zero-padded, lowercase hexadecimal encoding of n as a 16-byte slice.
func Hex64(n uint64) []byte {
	var dst [16]byte

	Hex64UB(n, &dst)

	return dst[:]
}

// Hex64UB writes the zero-padded, lowercase hexadecimal encoding of n into dst.
func Hex64UB(n uint64, dst *[16]byte) {
	dst[0] = hexTable[n>>60&0xf]
	dst[1] = hexTable[n>>56&0xf]
	dst[2] = hexTable[n>>52&0xf]
	dst[3] = hexTable[n>>48&0xf]
	dst[4] = hexTable[n>>44&0xf]
	dst[5] = hexTable[n>>40&0xf]
	dst[6] = hexTable[n>>36&0xf]
	dst[7] = hexTable[n>>32&0xf]
	dst[8] = hexTable[n>>28&0xf]
	dst[9] = hexTable[n>>24&0xf]
	dst[10] = hexTable[n>>20&0xf]
	dst[11] = hexTable[n>>16&0xf]
	dst[12] = hexTable[n>>12&0xf]
	dst[13] = hexTable[n>>8&0xf]
	dst[14] = hexTable[n>>4&0xf]
	dst[15] = hexTable[n&0xf]
}

// Hex64B returns the lowercase hexadecimal encoding of src as a 16-byte slice.
func Hex64B(src [8]byte) []byte {
	var dst [16]byte

	Hex64BB(src, &dst)

	return dst[:]
}

// Hex64BB writes the lowercase hexadecimal encoding of src into dst.
func Hex64BB(src [8]byte, dst *[16]byte) {
	dst[0] = hexTable[src[0]>>4]
	dst[1] = hexTable[src[0]&0xf]
	dst[2] = hexTable[src[1]>>4]
	dst[3] = hexTable[src[1]&0xf]
	dst[4] = hexTable[src[2]>>4]
	dst[5] = hexTable[src[2]&0xf]
	dst[6] = hexTable[src[3]>>4]
	dst[7] = hexTable[src[3]&0xf]
	dst[8] = hexTable[src[4]>>4]
	dst[9] = hexTable[src[4]&0xf]
	dst[10] = hexTable[src[5]>>4]
	dst[11] = hexTable[src[5]&0xf]
	dst[12] = hexTable[src[6]>>4]
	dst[13] = hexTable[src[6]&0xf]
	dst[14] = hexTable[src[7]>>4]
	dst[15] = hexTable[src[7]&0xf]
}

// Hex32 returns the zero-padded, lowercase hexadecimal encoding of n as an 8-byte slice.
func Hex32(n uint32) []byte {
	var dst [8]byte

	Hex32UB(n, &dst)

	return dst[:]
}

// Hex32UB writes the zero-padded, lowercase hexadecimal encoding of n into dst.
func Hex32UB(n uint32, dst *[8]byte) {
	dst[0] = hexTable[n>>28&0xf]
	dst[1] = hexTable[n>>24&0xf]
	dst[2] = hexTable[n>>20&0xf]
	dst[3] = hexTable[n>>16&0xf]
	dst[4] = hexTable[n>>12&0xf]
	dst[5] = hexTable[n>>8&0xf]
	dst[6] = hexTable[n>>4&0xf]
	dst[7] = hexTable[n&0xf]
}

// Hex32B returns the lowercase hexadecimal encoding of src as an 8-byte slice.
func Hex32B(src [4]byte) []byte {
	var dst [8]byte

	Hex32BB(src, &dst)

	return dst[:]
}

// Hex32BB writes the lowercase hexadecimal encoding of src into dst.
func Hex32BB(src [4]byte, dst *[8]byte) {
	dst[0] = hexTable[src[0]>>4]
	dst[1] = hexTable[src[0]&0xf]
	dst[2] = hexTable[src[1]>>4]
	dst[3] = hexTable[src[1]&0xf]
	dst[4] = hexTable[src[2]>>4]
	dst[5] = hexTable[src[2]&0xf]
	dst[6] = hexTable[src[3]>>4]
	dst[7] = hexTable[src[3]&0xf]
}

// Hex16 returns the zero-padded, lowercase hexadecimal encoding of n as a 4-byte slice.
func Hex16(n uint16) []byte {
	var dst [4]byte

	Hex16UB(n, &dst)

	return dst[:]
}

// Hex16UB writes the zero-padded, lowercase hexadecimal encoding of n into dst.
func Hex16UB(n uint16, dst *[4]byte) {
	dst[0] = hexTable[n>>12&0xf]
	dst[1] = hexTable[n>>8&0xf]
	dst[2] = hexTable[n>>4&0xf]
	dst[3] = hexTable[n&0xf]
}

// Hex16B returns the lowercase hexadecimal encoding of src as a 4-byte slice.
func Hex16B(src [2]byte) []byte {
	var dst [4]byte

	Hex16BB(src, &dst)

	return dst[:]
}

// Hex16BB writes the lowercase hexadecimal encoding of src into dst.
func Hex16BB(src [2]byte, dst *[4]byte) {
	dst[0] = hexTable[src[0]>>4]
	dst[1] = hexTable[src[0]&0xf]
	dst[2] = hexTable[src[1]>>4]
	dst[3] = hexTable[src[1]&0xf]
}

// Hex8 returns the zero-padded, lowercase hexadecimal encoding of n as a 2-byte slice.
func Hex8(n uint8) []byte {
	var dst [2]byte

	Hex8UB(n, &dst)

	return dst[:]
}

// Hex8UB writes the zero-padded, lowercase hexadecimal encoding of n into dst.
func Hex8UB(n uint8, dst *[2]byte) {
	dst[0] = hexTable[n>>4&0xf]
	dst[1] = hexTable[n&0xf]
}

// Hex8B returns the lowercase hexadecimal encoding of src as a 2-byte slice.
func Hex8B(src [1]byte) []byte {
	var dst [2]byte

	Hex8BB(src, &dst)

	return dst[:]
}

// Hex8BB writes the lowercase hexadecimal encoding of src into dst.
func Hex8BB(src [1]byte, dst *[2]byte) {
	dst[0] = hexTable[src[0]>>4]
	dst[1] = hexTable[src[0]&0xf]
}
