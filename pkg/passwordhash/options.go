package passwordhash

// Option applies a configuration change to [Params].
type Option func(*Params)

// WithKeyLen sets the derived key length (Tag length) in bytes.
// The value is clamped to [16, 1024]: 16 bytes (128 bits) is a safe floor, as
// shorter keys are trivially brute-forceable offline, and 1024 bytes is the
// largest length the verification path accepts; a longer key would mint a hash
// that could never be verified. The default is 32 bytes. (Hashes stored with a
// shorter key remain verifiable for backward compatibility.)
func WithKeyLen(v uint32) Option {
	return func(ph *Params) {
		ph.KeyLen = min(max(minHashKeyLen, v), maxVerifyKeyLen)
	}
}

// WithSaltLen sets random salt length (Nonce S) in bytes.
// The value is clamped to [8, 1024]: 8 bytes (64 bits) is a conservative floor
// for rainbow-table and pre-computation resistance, and 1024 bytes is the
// largest length the verification path accepts. The recommended and default
// value is 16 bytes. (Hashes stored with a shorter salt remain verifiable for
// backward compatibility.)
func WithSaltLen(v uint32) Option {
	return func(ph *Params) {
		ph.SaltLen = min(max(minHashSaltLen, v), maxVerifySaltLen)
	}
}

// WithTime sets the Argon2id time cost: the number of passes over memory.
// The value is clamped to [1, 1024]. Higher values increase resistance to
// brute-force attacks at the cost of hashing latency; 1024 is the largest value
// the verification path accepts, so it is the effective ceiling. OWASP
// recommends tuning so that hashing takes 0.5 to 1 s on target hardware.
func WithTime(v uint32) Option {
	return func(ph *Params) {
		ph.Time = min(max(minTime, v), maxVerifyTime)
	}
}

// WithMemory sets Argon2 memory cost in KiB.
// The value is clamped to [8, 4194304] (up to 4 GiB); 4 GiB is the largest the
// verification path accepts, so it is the effective ceiling. The actual number
// of blocks is m', which is m rounded down to the nearest multiple of 4*p.
func WithMemory(v uint32) Option {
	return func(ph *Params) {
		ph.Memory = min(max(minMemory, v), maxVerifyMemory)
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
// The value is clamped to a minimum of 1 so the length guard can never be
// silently disabled by passing 0.
func WithMinPasswordLength(v uint32) Option {
	return func(ph *Params) {
		ph.minPLen = max(minPasswordLength, v)
	}
}

// WithMaxPasswordLength sets the maximum accepted password length in bytes.
// Passwords longer than this are rejected before hashing, preventing
// denial-of-service attacks via extremely long input strings.
// If the value is lower than the configured minimum length, [New] raises it to
// that minimum so the accepted-length window is never empty.
func WithMaxPasswordLength(v uint32) Option {
	return func(ph *Params) {
		ph.maxPLen = v
	}
}

// WithVerifyCostMultiplier bounds how much more expensive a stored hash may be
// to verify than this configuration's own cost. At verification the time and
// memory parameters embedded in a stored hash are accepted only up to this
// multiple of the configured [WithTime] and [WithMemory] values (never beyond
// the absolute ceilings of 1024 passes and 4 GiB); a blob demanding more is
// rejected as [ErrInvalidHashData] before any Argon2 computation.
//
// The default is 4, which leaves room for the ordinary "raise the configuration,
// then rehash on the next login" upgrade path: a freshly minted hash costs 1x
// and a legacy hash costs less than the current configuration, so neither is
// rejected, while stopping a single forged or corrupt row from pinning a
// verifier at up to 4 GiB and minutes of CPU per login attempt on that account.
//
// Lower it toward 1 where a stored hash could ever be attacker-controlled (a
// "verify this hash" API, a hash copied from another system) so the accepted
// band sits just above your real maximum cost. Raise it before a large single
// step increase in cost, or where hashes minted at a much higher cost must stay
// verifiable under a lower configuration. The value is clamped to a minimum of 1
// so the accepted band can never collapse to nothing.
func WithVerifyCostMultiplier(v uint32) Option {
	return func(ph *Params) {
		ph.verifyCostMultiplier = max(minVerifyCostMultiplier, v)
	}
}

// WithFormat selects the serialization emitted by [Params.PasswordHash] and the
// set of formats [Params.PasswordNeedsRehash] treats as current.
//
// f is the format newly minted hashes use: [FormatJSON] (the default
// self-describing base64 JSON) or [FormatPHC] (the cross-language PHC string
// format). Any other value is normalized to [FormatJSON].
//
// accept optionally lists additional formats that PasswordNeedsRehash reports
// as current; unknown values are ignored rather than aliased to a format the
// caller did not request. By default only f is accepted, so an existing store
// converges to f through the ordinary rehash-on-login flow: a stored hash in
// any other format is reported as needing a rehash even when its Argon2
// parameters match the configuration. Listing the other format keeps a
// deliberately mixed store with no forced convergence:
//
//	WithFormat(FormatPHC)             // emit PHC, converge existing hashes to PHC
//	WithFormat(FormatPHC, FormatJSON) // emit PHC, keep existing JSON hashes as-is
//
// Verification is unaffected: [Params.PasswordVerify] auto-detects and accepts
// every supported format regardless of this option, so no configuration can
// lock out an existing credential.
//
// It has no effect on the pepper-encrypted methods ([Params.EncryptPasswordHash]
// and friends): their output is an opaque AES-GCM ciphertext, so the inner
// serialization is never read by another system and there is nothing to make
// interoperable.
func WithFormat(f Format, accept ...Format) Option {
	return func(ph *Params) {
		ph.format = normalizeFormat(f)
		ph.acceptedFormats = formatBit(ph.format)

		for _, a := range accept {
			// Only supported values join the accepted set: aliasing an unknown value
			// to a real format would silently widen acceptance beyond what the
			// caller asked for, delaying convergence instead of failing safe.
			if a == FormatJSON || a == FormatPHC {
				ph.acceptedFormats |= formatBit(a)
			}
		}
	}
}
