package passwordhash

// Option is a type alias for a function that configures the password hasher.
type Option func(*Params)

// WithKeyLen overwrites the default length of the returned byte-slice that can be used as cryptographic key (Tag length).
// It must be an integer number of bytes from 4 to 2^(32)-1.
// The default value is 32 bytes.
func WithKeyLen(v uint32) Option {
	return func(ph *Params) {
		ph.KeyLen = max(minKeyLen, v)
	}
}

// WithSaltLen overwrites the default length of the random password salt (Nonce S).
// It must be not greater than 2^(32)-1 bytes.
// The value of 16 bytes is recommended for password hashing.
func WithSaltLen(v uint32) Option {
	return func(ph *Params) {
		ph.SaltLen = max(minSaltLen, v)
	}
}

// WithTime (t) is the default number of passes (iterations) over the memory.
// It must be an integer value from 1 to 2^(32)-1.
func WithTime(v uint32) Option {
	return func(ph *Params) {
		ph.Time = max(minTime, v)
	}
}

// WithMemory overwrites the default size of the memory in KiB.
// It must be an integer number of kibibytes from 8*p to 2^(32)-1.
// The actual number of blocks is m', which is m rounded down to the nearest multiple of 4*p.
func WithMemory(v uint32) Option {
	return func(ph *Params) {
		ph.Memory = max(minMemory, v)
	}
}

// WithThreads overwrites the default number ot threads (p)
// Threads represents the degree of parallelism p that determines how many independent
// (but	synchronizing) computational chains (lanes) can be run.
// According to the RFC9106 it must be an integer value from 1 to 2^(24)-1, but in this implementation is limited to 2^(8)-1.
func WithThreads(v uint8) Option {
	return func(ph *Params) {
		ph.Threads = max(minThreads, v)
	}
}

// WithMinPasswordLength overwrites the default maximum length of the input password (Message string P).
// It must have a length not greater than 2^(32)-1 bytes.
func WithMinPasswordLength(v uint32) Option {
	return func(ph *Params) {
		ph.minPLen = v
	}
}

// WithMaxPasswordLength overwrites the default maximum length of the input password (Message string P).
// It must have a length not greater than 2^(32)-1 bytes.
func WithMaxPasswordLength(v uint32) Option {
	return func(ph *Params) {
		ph.maxPLen = v
	}
}
