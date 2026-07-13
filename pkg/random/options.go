package random

import "bytes"

// Option is the interface that allows to set client options.
type Option func(c *Rnd)

// WithByteToCharMap customizes the byte-to-character mapping for RandString().
// Empty maps restore the default; maps > 256 bytes are truncated to 256.
//
// The map is copied, so the caller keeps ownership of the slice it passes in and
// may reuse, mutate, or zero it afterwards without affecting the generator.
//
// The map holds bytes, not runes, and [Rnd.RandString] draws one byte per output
// position, so entries must be single-byte (ASCII) values. A map containing
// multi-byte UTF-8 runes is not rejected, but each rune is split into its
// constituent bytes and those bytes are then drawn independently of one another,
// so the generated strings are almost always invalid UTF-8. Truncation at 256
// bytes can likewise split a trailing rune.
func WithByteToCharMap(cm []byte) Option {
	switch d := len(cm); {
	case d == 0:
		cm = []byte(chrMapDefault)
	case d > chrMapMaxLen:
		cm = cm[:chrMapMaxLen]
	}

	// Copy: retaining the caller's backing array would let a later mutation of it
	// silently reconfigure the generator (clearing the slice would turn RandString
	// into a NUL generator), and a concurrent one would be a data race.
	cm = bytes.Clone(cm)

	return func(c *Rnd) {
		c.chrMap = cm
	}
}

// WithFallbackHook registers fn to be called whenever the configured [io.Reader]
// fails and a non-failing helper ([Rnd.RandUint32], [Rnd.RandUint64],
// [Rnd.UUIDv7], and everything built on them) silently substitutes math/rand/v2
// for it.
//
// The substituted source is Go's OS-seeded ChaCha8 global generator, so the
// output is not predictable; the reason to observe the event is that the entropy
// source is no longer the one the caller configured, which matters for auditing
// and for alerting on a failing HSM- or KMS-backed reader. With the default
// [crypto/rand.Reader] the hook can never fire, because its Read cannot fail.
//
// fn is called synchronously on the generating goroutine, so it must be fast and
// must not call back into the same [Rnd]. It may be called concurrently from
// several goroutines and must therefore be safe for concurrent use.
func WithFallbackHook(fn func()) Option {
	return func(c *Rnd) {
		c.fallback = fn
	}
}
