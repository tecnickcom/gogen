/*
Package uidc provides functions to generate simple time-and-random-based unique identifiers.

It provides functions to generate 64 and 128 bit random identifiers in base-36 string format.
*/
package uidc

import (
	"strconv"
	"time"

	"github.com/Vonage/gosrvlib/pkg/random"
)

// NewID64 generates and returns a new base-36-string-formatted 64 bit unique ID based on time (high 32 bit) and a random number (low 32 bit).
// NOTE: the zero time is set to the 1st of January of 10 years ago.
func NewID64() string {
	t := time.Now().UTC()
	offset := time.Date(t.Year()-10, 1, 1, 0, 0, 0, 0, time.UTC).Unix() // [s] time starts 1st JAN 10 years ago

	return strconv.FormatUint((((uint64)(t.Unix()-offset))<<32)+(uint64)(random.New(nil).RandUint32()), 36)
}

// NewID128 generates and returns a new base-36-string-formatted 128 bit unique ID based on time (high 64 bit) and a random number (low 64 bit).
func NewID128() string {
	return strconv.FormatUint((uint64)(time.Now().UTC().UnixNano()), 36) + strconv.FormatUint(random.New(nil).RandUint64(), 36)
}
