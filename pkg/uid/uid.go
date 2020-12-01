// Package uid provides fast functions to generate simple time-and-random-based unique identifiers.
package uid

import (
	"encoding/binary"
	"math/rand"
	"strconv"
	"time"
)

// InitRandSeed initialize the random generator seed.
// You have to call this first and only once.
func InitRandSeed() error {
	var b [8]byte
	_, err := rand.Read(b[:]) // #nosec
	if err == nil {
		rand.Seed(int64(binary.LittleEndian.Uint64(b[:])))
	}
	return err
}

// NewID64 generates and return a new base-36-string-formatted 64 bit unique ID based on time (high 32 bit) and a random number (low 32 bit).
// NOTE: the zero time is set to the 1st of january of 10 year ago.
func NewID64() string {
	t := time.Now().UTC()
	offset := time.Date(t.Year()-10, 1, 1, 0, 0, 0, 0, time.UTC).Unix()                      // [s] time starts 1st JAN 10 years ago
	return strconv.FormatUint((((uint64)(t.Unix()-offset))<<32)+(uint64)(rand.Uint32()), 36) // #nosec
}

// NewID128 generates and return a new base-36-string-formatted 128 bit unique ID based on time (high 64 bit) and a random number (low 64 bit).
func NewID128() string {
	return strconv.FormatUint((uint64)(time.Now().UTC().UnixNano()), 36) + strconv.FormatUint(rand.Uint64(), 36) // #nosec
}
