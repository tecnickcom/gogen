// Package uidc provides cryptographic-grade functions to generate simple time-and-random-based unique identifiers.
package uidc

import (
	"crypto/rand"
	"encoding/binary"
	"strconv"
	"time"
)

// NewID64 generates and return a new base-36-string-formatted 64 bit unique ID based on time (high 32 bit) and a random number (low 32 bit).
// NOTE: the zero time is set to the 1st of january of 10 year ago.
func NewID64() string {
	t := time.Now().UTC()
	offset := time.Date(t.Year()-10, 1, 1, 0, 0, 0, 0, time.UTC).Unix() // [s] time starts 1st JAN 10 years ago
	b := make([]byte, 4)                                                // 4 bytes for 32 bit
	_, _ = rand.Read(b)
	return strconv.FormatUint((((uint64)(t.Unix()-offset))<<32)+(uint64)(binary.LittleEndian.Uint32(b)), 36)
}

// NewID128 generates and return a new base-36-string-formatted 128 bit unique ID based on time (high 64 bit) and a random number (low 64 bit).
func NewID128() string {
	b := make([]byte, 8) // 8 bytes for 64 bit
	_, _ = rand.Read(b)
	return strconv.FormatUint((uint64)(time.Now().UTC().UnixNano()), 36) + strconv.FormatUint(binary.LittleEndian.Uint64(b), 36)
}
