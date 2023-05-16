// Package enumbitmap provides functions to encode slices of enumeration strings into integer bitmap values and vice versa.
// Each bit correspond to a unique enumeration value.
//
// Example:
//
//	00000000 =   0 dec = NONE
//	00000001 =   1 dec = FIRST
//	00000010 =   2 dec = SECOND
//	00000100 =   4 dec = THIRD
//	00001000 =   8 dec = FOURTH
//	00010000 =  16 dec = FIFTH
//	00100000 =  32 dec = SIXTH
//	01000000 =  64 dec = SEVENTH
//	10000000 = 128 dec = EIGHTH
//	00001001 = 1 + 8 = 9 dec = FIRST + FOURTH
package enumbitmap

import (
	"fmt"
)

const (
	// maxBit is the maximum supported number of bits.
	// It is also the maximum number of items that can be represented with a single integer.
	maxBit = 32
)

// BitMapToStrings converts a int bitmap value into a string slice.
func BitMapToStrings(enum map[int]string, v int) ([]string, error) {
	if v == 0 {
		return []string{}, nil
	}

	s := make([]string, 0, maxBit)
	errBits := make([]int, 0, maxBit)

	i := 1

	for bit := 1; bit <= maxBit; bit++ {
		if v&i == i {
			name, ok := enum[i]
			if ok {
				s = append(s, name)
			} else {
				errBits = append(errBits, bit)
			}
		}

		i = (i << 1)
	}

	var err error

	if len(errBits) > 0 {
		err = fmt.Errorf("unknown bit values: %v", errBits)
	}

	return s, err
}

// StringsToBitMap converts a string slice into a int bitmap value.
func StringsToBitMap(enum map[string]int, s []string) (int, error) {
	errStrings := make([]string, 0, maxBit)

	var v int

	for _, key := range s {
		id, ok := enum[key]
		if ok {
			v |= id
		} else {
			errStrings = append(errStrings, key)
		}
	}

	var err error

	if len(errStrings) > 0 {
		err = fmt.Errorf("unknown string values: %q", errStrings)
	}

	return v, err
}
