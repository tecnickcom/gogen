/*
Package uid provides functions to generate simple time-and-random-based unique
identifiers.

Deprecated: Use github.com/Vonage/gosrvlib/pkg/uidc instead.
*/
package uid

import (
	"github.com/Vonage/gosrvlib/pkg/uidc"
)

// NewID64 generates and returns a new base-36-string-formatted 64 bit unique ID based on time (high 32 bit) and a random number (low 32 bit).
// NOTE: the zero time is set to the 1st of January of 10 years ago.
// Deprecated: Use github.com/Vonage/gosrvlib/pkg/uidc NewID64() instead.
func NewID64() string {
	return uidc.NewID64()
}

// NewID128 generates and returns a new base-36-string-formatted 128 bit unique ID based on time (high 64 bit) and a random number (low 64 bit).
// Deprecated: Use github.com/Vonage/gosrvlib/pkg/uidc NewID128() instead.
func NewID128() string {
	return uidc.NewID128()
}
