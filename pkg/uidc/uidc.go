/*
Package uidc provides functions to generate simple time-and-random-based unique
identifiers.

It offers functions to generate 64-bit and 128-bit random identifiers in base-36
string format.

The generated IDs are not intended for cryptographic or security-related
purposes. Instead, they serve as simple unique identifiers for various use
cases.

Deprecated: Please use github.com/tecnickcom/gogen/pkg/random instead.
*/
package uidc

import (
	"github.com/tecnickcom/gogen/pkg/random"
)

var rnd = random.New(nil) //nolint:gochecknoglobals

// NewID64 returns a 64-bit unique ID encoded as a base-36 string.
// The upper 32 bits contain a timestamp and the lower 32 bits are random.
// The timestamp epoch is January 1st of the current decade's starting year.
//
// Deprecated: use random.UID64().String() instead.
func NewID64() string {
	return rnd.UID64().String()
}

// NewID128 returns a 128-bit unique identifier encoded as a base-36 string.
// The high 64 bits are derived from the current time; the low 64 bits are random.
//
// Deprecated: use random.UID128().String() instead.
func NewID128() string {
	return rnd.UID128().String()
}
