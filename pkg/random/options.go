package random

// Option is the interface that allows to set client options.
type Option func(c *Rnd)

// WithByteToCharMap overwrites the default slice used to map random bytes to characters.
// If cm is empty, then the default character map will be used.
// The maximum cm length is 256, if it is greater than 256, it will be truncated.
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
