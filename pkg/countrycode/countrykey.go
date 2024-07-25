package countrycode

import (
	"errors"
	"fmt"
	"strconv"
)

const (
	bitLenChar int = 5 // 5 bit per character.
)

// Binary length of each CountryKey section.
const (
	bitLenTLD       int = 2 * bitLenChar // 2 characters, 5 bit per character.
	bitLenIntRegion int = 5              // max 2^5 = 32 distinct values.
	bitLenSubRegion int = 5              // max 2^5 = 32 distinct values.
	bitLenRegion    int = 5              // max 2^5 = 32 distinct values.
	bitLenNumeric   int = 10             // max 3 numerical digits: log2(999) ~> 10.
	bitLenAlpha3    int = 3 * bitLenChar // 3 characters, 5 bit per character.
	bitLenAlpha2    int = 2 * bitLenChar // 2 characters, 5 bit per character.
	bitLenStatus    int = 3              // max 2^3 = 8 distinct values.
)

// Binary bit positions for when each countrykey element starts (counting from the right - LSB).
const (
	bitPosTLD       int = 0                                 // 0.
	bitPosIntRegion int = bitPosTLD + bitLenTLD             // 10.
	bitPosSubRegion int = bitPosIntRegion + bitLenIntRegion // 15.
	bitPosRegion    int = bitPosSubRegion + bitLenSubRegion // 20.
	bitPosNumeric   int = bitPosRegion + bitLenRegion       // 25.
	bitPosAlpha3    int = bitPosNumeric + bitLenNumeric     // 35.
	bitPosAlpha2    int = bitPosAlpha3 + bitLenAlpha3       // 50.
	bitPosStatus    int = bitPosAlpha2 + bitLenAlpha2       // 60.
)

// Binary bit masks for when each CountryKey element starts and ends.
const (
	// -------------------------32109876 54321098 76543210 98765432 10987654 32109876 54321098 76543210 // 64 bit.
	bitMaskTLD       uint64 = 0b00000000_00000000_00000000_00000000_00000000_00000000_00000011_11111111 // 10 bit // pos  0.
	bitMaskIntRegion uint64 = 0b00000000_00000000_00000000_00000000_00000000_00000000_01111100_00000000 //  5 bit // pos 10.
	bitMaskSubRegion uint64 = 0b00000000_00000000_00000000_00000000_00000000_00001111_10000000_00000000 //  5 bit // pos 15.
	bitMaskRegion    uint64 = 0b00000000_00000000_00000000_00000000_00000001_11110000_00000000_00000000 //  5 bit // pos 20.
	bitMaskNumeric   uint64 = 0b00000000_00000000_00000000_00000111_11111110_00000000_00000000_00000000 // 10 bit // pos 25.
	bitMaskAlpha3    uint64 = 0b00000000_00000011_11111111_11111000_00000000_00000000_00000000_00000000 // 15 bit // pos 35.
	bitMaskAlpha2    uint64 = 0b00001111_11111100_00000000_00000000_00000000_00000000_00000000_00000000 // 10 bit // pos 50.
	bitMaskStatus    uint64 = 0b01110000_00000000_00000000_00000000_00000000_00000000_00000000_00000000 //  3 bit // pos 60.
)

// Binary bit masks for when each character in the alpha-2, alpha-3 and TLD codes starts and ends.
const (
	bitMaskChar0 uint16 = 0b0_00000_00000_11111 // 5 bit // pos  0.
	bitMaskChar1 uint16 = 0b0_00000_11111_00000 // 5 bit // pos  5.
	bitMaskChar2 uint16 = 0b0_11111_00000_00000 // 5 bit // pos 10.
)

const (
	mask03Bit = 0b00000111
	mask05Bit = 0b00011111
	mask10Bit = 0b00000011_11111111
	mask15Bit = 0b01111111_11111111
)

const (
	bitMaskValStatus    = mask03Bit
	bitMaskValApha2     = mask10Bit
	bitMaskValAlpha3    = mask15Bit
	bitMaskValNumeric   = mask10Bit
	bitMaskValRegion    = mask05Bit
	bitMaskValSubRegion = mask05Bit
	bitMaskValIntRegion = mask05Bit
	bitMaskValTLD       = mask10Bit
)

// Binary bit positions for when each character in the alpha-2, alpha-3 and TLD codes starts (counting from the right - LSB).
const (
	bitPosChar0 int = 0
	bitPosChar1 int = bitPosChar0 + bitLenChar
	bitPosChar2 int = bitPosChar1 + bitLenChar
)

const (
	chrOffsetUpper uint16 = ('A' - 1) // 64.
	chrOffsetLower uint16 = ('a' - 1) // 96.
)

var (
	errInvalidKey       = errors.New("invalid key")
	errInvalidLength    = errors.New("invalid code length")
	errInvalidCharacter = errors.New("invalid code, it must contain only A-Z characters")
)

// countryKeyElem represent CountryKey,
// a country data binary encoding in uint64 format.
type countryKeyElem struct {
	status    uint8  // 3 bit status
	alpha2    uint16 // 10 bit alpha2
	alpha3    uint16 // 15 bit alpha3
	numeric   uint16 // 10 bit numeric
	region    uint8  // 5 bit region
	subregion uint8  // 5 bit sub-region
	intregion uint8  // 5 bit intermediate-region
	tld       uint16 // 10 bit TLD
}

func decodeCountryKey(key uint64) *countryKeyElem {
	return &countryKeyElem{
		status:    uint8((key & bitMaskStatus) >> bitPosStatus),
		alpha2:    uint16((key & bitMaskAlpha2) >> bitPosAlpha2),
		alpha3:    uint16((key & bitMaskAlpha3) >> bitPosAlpha3),
		numeric:   uint16((key & bitMaskNumeric) >> bitPosNumeric),
		region:    uint8((key & bitMaskRegion) >> bitPosRegion),
		subregion: uint8((key & bitMaskSubRegion) >> bitPosSubRegion),
		intregion: uint8((key & bitMaskIntRegion) >> bitPosIntRegion),
		tld:       uint16((key & bitMaskTLD) >> bitPosTLD),
	}
}

// encodeCountryKey encodes the country data into a uint64.
// The returned Country key is an RNCK, see:
// "Reversible Numeric Composite Key (RNCK), N Asuni - arXiv preprint arXiv:2306.04353, 2023".
func (e *countryKeyElem) encodeCountryKey() uint64 {
	return ((uint64(e.status&bitMaskValStatus) << bitPosStatus) |
		(uint64(e.alpha2&bitMaskValApha2) << bitPosAlpha2) |
		(uint64(e.alpha3&bitMaskValAlpha3) << bitPosAlpha3) |
		(uint64(e.numeric&bitMaskValNumeric) << bitPosNumeric) |
		(uint64(e.region&bitMaskValRegion) << bitPosRegion) |
		(uint64(e.subregion&bitMaskValSubRegion) << bitPosSubRegion) |
		(uint64(e.intregion&bitMaskValIntRegion) << bitPosIntRegion) |
		(uint64(e.tld&bitMaskValTLD) << bitPosTLD))
}

func charOffset(b byte, offset uint16) (uint16, error) {
	c := (uint16(b) - offset)
	if c < 1 || c > 26 { // A-Z or a-z
		return 0, errInvalidCharacter
	}

	return c, nil
}

func charOffsetUpper(b byte) (uint16, error) {
	return charOffset(b, chrOffsetUpper)
}

func charOffsetLower(b byte) (uint16, error) {
	return charOffset(b, chrOffsetLower)
}

// encodeAlpha2 encodes a 2-uppercase-character alpha-2 code (AA to ZZ) into a uint16.
func encodeAlpha2(s string) (uint16, error) {
	if len(s) != 2 {
		return 0, errInvalidLength
	}

	c1, err := charOffsetUpper(s[0])
	if err != nil {
		return c1, err
	}

	c0, err := charOffsetUpper(s[1])
	if err != nil {
		return c0, err
	}

	return ((c1 << bitPosChar1) | (c0 << bitPosChar0)), nil
}

// decodeAlpha2 decodes a uint16 into a 2-uppercase-character alpha-2 code.
func decodeAlpha2(code uint16) string {
	return string([]byte{
		byte(((code & bitMaskChar1) >> bitPosChar1) + chrOffsetUpper),
		byte(((code & bitMaskChar0) >> bitPosChar0) + chrOffsetUpper),
	})
}

func encodeAlpha3(s string) (uint16, error) {
	if len(s) != 3 {
		return 0, errInvalidLength
	}

	c2, err := charOffsetUpper(s[0])
	if err != nil {
		return c2, err
	}

	c1, err := charOffsetUpper(s[1])
	if err != nil {
		return c1, err
	}

	c0, err := charOffsetUpper(s[2])
	if err != nil {
		return c0, err
	}

	return ((c2 << bitPosChar2) | (c1 << bitPosChar1) | (c0 << bitPosChar0)), nil
}

func decodeAlpha3(code uint16) string {
	return string([]byte{
		byte(((code & bitMaskChar2) >> bitPosChar2) + chrOffsetUpper),
		byte(((code & bitMaskChar1) >> bitPosChar1) + chrOffsetUpper),
		byte(((code & bitMaskChar0) >> bitPosChar0) + chrOffsetUpper),
	})
}

func encodeTLD(s string) (uint16, error) {
	if len(s) != 2 {
		return 0, errInvalidLength
	}

	c1, err := charOffsetLower(s[0])
	if err != nil {
		return c1, err
	}

	c0, err := charOffsetLower(s[1])
	if err != nil {
		return c0, err
	}

	return ((c1 << bitPosChar1) | (c0 << bitPosChar0)), nil
}

func decodeTLD(code uint16) string {
	return string([]byte{
		byte(((code & bitMaskChar1) >> bitPosChar1) + chrOffsetLower),
		byte(((code & bitMaskChar0) >> bitPosChar0) + chrOffsetLower),
	})
}

func encodeNumeric(s string) (uint16, error) {
	if len(s) != 3 {
		return 0, errInvalidLength
	}

	v, err := strconv.ParseUint(s, 10, 16)
	if err != nil {
		return 0, fmt.Errorf("invalid numeric code: %w", err)
	}

	return uint16(v), nil
}
