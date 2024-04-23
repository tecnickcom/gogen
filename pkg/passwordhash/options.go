package passwordhash

// Option is a type alias for a function that configures the password hasher.
type Option func(*Params)

// WithKeyLen overwrites the default length of the returned byte-slice that can be used as cryptographic key.
// The default value is 32 bytes.
func WithKeyLen(v uint32) Option {
	return func(ph *Params) {
		ph.KeyLen = v
	}
}

// WithSaltLen overwrites the default length of the random password salt.
// The default value is 16 bytes.
func WithSaltLen(v uint32) Option {
	return func(ph *Params) {
		ph.SaltLen = v
	}
}

// WithTime overwrites the number of passes over the memory.
// The default value is 1.
func WithTime(v uint32) Option {
	return func(ph *Params) {
		ph.Time = v
	}
}

// WithMemory overwrites the size of the memory in KiB.
// The default value is 65_536 KiB.
func WithMemory(v uint32) Option {
	return func(ph *Params) {
		ph.Memory = v
	}
}

// WithThreads overwrites the number of threads used by the algorithm.
// The default value is the number of available CPUs.
func WithThreads(v uint8) Option {
	return func(ph *Params) {
		ph.Threads = v
	}
}
