package passwordhash

// Option applies a configuration change to [Params].
type Option func(*Params)

// WithKeyLen sets the derived key length (Tag length) in bytes.
// It must be an integer number of bytes from 4 to 2^(32)-1.
// The default value is 32 bytes.
func WithKeyLen(v uint32) Option {
	return func(ph *Params) {
		ph.KeyLen = max(minKeyLen, v)
	}
}

// WithSaltLen sets random salt length (Nonce S) in bytes.
// It must be not greater than 2^(32)-1 bytes.
// The value of 16 bytes is recommended for password hashing.
func WithSaltLen(v uint32) Option {
	return func(ph *Params) {
		ph.SaltLen = max(minSaltLen, v)
	}
}

// WithTime sets the Argon2id time cost: the number of passes over memory.
// Higher values increase resistance to brute-force attacks at the cost of
// hashing latency. Must be >= 1 (minimum enforced automatically).
// OWASP recommends tuning so that hashing takes 0.5–1 s on target hardware.
func WithTime(v uint32) Option {
	return func(ph *Params) {
		ph.Time = max(minTime, v)
	}
}

// WithMemory sets Argon2 memory cost in KiB.
// It must be an integer number of kibibytes from 8*p to 2^(32)-1.
// The actual number of blocks is m', which is m rounded down to the nearest multiple of 4*p.
func WithMemory(v uint32) Option {
	return func(ph *Params) {
		ph.Memory = max(minMemory, v)
	}
}

// WithThreads sets Argon2 parallelism (threads/lane count).
// It controls the degree of parallelism p, which determines how many independent
// (but synchronizing) computational chains (lanes) can be run.
// According to RFC 9106, valid values are 1 to 2^(24)-1; this implementation
// limits the value to 2^(8)-1.
func WithThreads(v uint8) Option {
	return func(ph *Params) {
		ph.Threads = max(minThreads, v)
	}
}

// WithMinPasswordLength sets the minimum accepted password length in bytes.
// Passwords shorter than this are rejected by [Params.PasswordHash] before
// any CPU-intensive computation, enforcing password policy at zero cost.
func WithMinPasswordLength(v uint32) Option {
	return func(ph *Params) {
		ph.minPLen = v
	}
}

// WithMaxPasswordLength sets the maximum accepted password length in bytes.
// Passwords longer than this are rejected before hashing, preventing
// denial-of-service attacks via extremely long input strings.
func WithMaxPasswordLength(v uint32) Option {
	return func(ph *Params) {
		ph.maxPLen = v
	}
}
