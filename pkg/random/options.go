package random

// Option is the interface that allows to set client options.
type Option func(c *Rnd)

// WithByteToCharMap customizes the byte-to-character mapping for RandString().
// Empty maps restore the default; maps > 256 bytes are truncated to 256.
func WithByteToCharMap(cm []byte) Option {
	switch d := len(cm); {
	case d == 0:
		cm = []byte(chrMapDefault)
	case d > chrMapMaxLen:
		cm = cm[:chrMapMaxLen]
	}

	return func(c *Rnd) {
		c.chrMap = cm
	}
}
