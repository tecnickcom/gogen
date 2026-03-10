/*
Package uhex contains a collection of fast-encoding hex functions for unsigned integers.
*/
package uhex

const hexTable = "0123456789abcdef"

// Hex64 converts a uint64 to 16-byte hexadecimal representation.
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

// Hex32 converts a uint32 to 8-byte hexadecimal representation.
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

// Hex16 converts a uint16 to 4-byte hexadecimal representation.
func Hex16(n uint16) []byte {
	var buf [4]byte

	buf[0] = hexTable[n>>12&0xf]
	buf[1] = hexTable[n>>8&0xf]
	buf[2] = hexTable[n>>4&0xf]
	buf[3] = hexTable[n&0xf]

	return buf[:]
}

// Hex8 converts a uint8 to 2-byte hexadecimal representation.
func Hex8(n uint8) []byte {
	var buf [2]byte

	buf[0] = hexTable[n>>4&0xf]
	buf[1] = hexTable[n&0xf]

	return buf[:]
}
