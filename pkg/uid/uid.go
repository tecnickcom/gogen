// Package uid provides fast functions to generate simple time-and-random-based unique identifiers.
// Deprecated: use github.com/Vonage/gosrvlib/pkg/uidc package instead.
package uid

import "github.com/Vonage/gosrvlib/pkg/uidc"

// InitRandSeed initialize the random generator seed.
// You have to call this first and only once.
// Deprecated: use github.com/Vonage/gosrvlib/pkg/uidc package instead.
func InitRandSeed() error {
	return nil
}

// NewID64 generates and return a new base-36-string-formatted 64 bit unique ID based on time (high 32 bit) and a random number (low 32 bit).
// NOTE: the zero time is set to the 1st of january of 10 year ago.
// Deprecated: use github.com/Vonage/gosrvlib/pkg/uidc package instead.
func NewID64() string {
	return uidc.NewID64()
}

// NewID128 generates and return a new base-36-string-formatted 128 bit unique ID based on time (high 64 bit) and a random number (low 64 bit).
// Deprecated: use github.com/Vonage/gosrvlib/pkg/uidc package instead.
func NewID128() string {
	return uidc.NewID128()
}
