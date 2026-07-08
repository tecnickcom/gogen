/*
Package passwordhash provides OWASP-compliant password hashing and verification
using the Argon2id algorithm (RFC 9106), with an optional AES-GCM encryption
layer (peppered hashing) for defense in depth.

# Problem

Storing passwords securely is one of the most critical and most frequently
mishandled tasks in application development. MD5, SHA-1, and even bcrypt are
either broken or insufficient against modern GPU-based attacks. Choosing the
correct algorithm, tuning its parameters, generating a cryptographically random
salt, and encoding everything into a portable, self-describing format — all
without introducing subtle timing-attack vulnerabilities — requires deep
cryptographic knowledge most teams would rather not reinvent.

# Solution

This package encapsulates the full OWASP Password Storage Cheat Sheet
(https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html)
recommendations into three method pairs on a single [Params] configuration object:

	p := passwordhash.New() // sensible OWASP defaults

	// Hash a password for storage.
	hash, err := p.PasswordHash(plaintext)

	// Verify a login attempt.
	ok, err := p.PasswordVerify(plaintext, hash)

	// After a successful verification, detect hashes minted with outdated
	// parameters so they can be transparently re-hashed and stored again.
	upgrade, err := p.PasswordNeedsRehash(hash)

For deployments that store a secret pepper outside the database, the encrypted
variants add an AES-GCM layer on top of the Argon2id hash:

	hash, err := p.EncryptPasswordHash(pepper, plaintext)
	ok, err  := p.EncryptPasswordVerify(pepper, plaintext, hash)

# Storage Format

The hashed password is stored as a base64-encoded JSON object that is fully
self-describing: it embeds the algorithm name, version, all Argon2id tuning
parameters, the random salt, and the derived key. This makes the stored value
portable across languages and systems, and allows parameters to be upgraded
without invalidating existing hashes.

Example JSON (before base64 encoding):

	{
	  "P": {
	    "A": "argon2id",  // algorithm name (always "argon2id")
	    "V": 19,          // Argon2id version (0x13)
	    "K": 32,          // derived key length in bytes
	    "S": 16,          // salt length in bytes
	    "T": 3,           // time cost (passes over memory)
	    "M": 65536,       // memory cost in KiB
	    "P": 4            // parallelism (threads)
	  },
	  "S": "wQYm4bfktbHq2omIwFu+4Q==",                       // base64 random salt
	  "K": "aU8hO900Odq6aKtWiWz3RW9ygn734liJaPtM6ynvkYI="   // base64 Argon2id hash
	}

The final stored value is the base64 encoding of the above JSON (~200 bytes).

# Features

  - Argon2id algorithm: resists both side-channel (Argon2i) and GPU-based
    (Argon2d) attacks — the current OWASP top recommendation (RFC 9106,
    https://www.rfc-editor.org/info/rfc9106).
  - Cryptographically random salt: a fresh salt is generated for every hash,
    preventing rainbow-table and pre-computation attacks.
  - Constant-time comparison: final hash comparison uses [crypto/subtle],
    preventing timing side-channel attacks.
  - Self-describing storage format: algorithm, version, and all parameters
    travel with the hash; no separate migration table is needed when tuning
    changes.
  - Optional pepper encryption: [Params.EncryptPasswordHash] and
    [Params.EncryptPasswordVerify] wrap the Argon2id hash in AES-GCM using a
    key stored outside the database, so a DB leak alone is insufficient to
    mount an offline attack.
  - Input length guards: the configured minimum length is enforced when
    hashing (registration policy) and the maximum length is enforced
    everywhere, rejecting oversized passwords before any CPU-intensive
    computation; oversized stored hash strings are likewise rejected before
    any decoding, preventing denial-of-service via extremely long inputs.
  - Transparent parameter upgrades: [Params.PasswordNeedsRehash] and
    [Params.EncryptPasswordNeedsRehash] detect hashes minted with outdated
    parameters so they can be re-hashed on the next successful login.
  - Sentinel errors: every failure class is matchable with [errors.Is] —
    policy rejection, invalid configuration, corrupt or forged hash data,
    algorithm/version mismatch, and invalid pepper key.
  - Tunable via functional options: [WithTime], [WithMemory], [WithThreads],
    [WithKeyLen], [WithSaltLen], [WithMinPasswordLength], and
    [WithMaxPasswordLength] let each deployment match the OWASP parameter
    guidance for its hardware profile.
  - Portable format: JSON + base64 storage can be decoded by any language with
    standard libraries, enabling cross-platform password verification.

# Verification Flow

 1. Reject the stored string before any decoding if it exceeds the maximum
    accepted size (16 KiB), bounding the cost of untrusted input.
 2. Decode the stored base64 string to retrieve the JSON object.
 3. Unmarshal the JSON to recover algorithm, version, parameters, and salt.
 4. Validate that the embedded parameters are within accepted bounds and
    internally consistent (the salt and key byte lengths match their declared
    sizes), rejecting forged or corrupted blobs before any computation.
 5. Validate that the stored algorithm and version match the library.
 6. Re-derive the key from the candidate password using the stored parameters
    and salt.
 7. Compare the derived key against the stored key with [crypto/subtle.ConstantTimeCompare]
    to prevent timing attacks.

# Parameter Tuning

The defaults (T=3, M=64 MiB, P=4) match the second recommended option set of
RFC 9106 §4 (https://datatracker.ietf.org/doc/html/rfc9106#section-4).
Parallelism is a flat constant, deliberately not derived from runtime.NumCPU():
Argon2 lanes are goroutines, so p=4 is valid on any host, and a
machine-independent default keeps the work factor reproducible across
heterogeneous fleets — with a per-machine default, hosts with different core
counts would mint hashes with different parameters and
[Params.PasswordNeedsRehash] would report an upgrade on every alternating
login, re-hashing forever.
For production deployments, benchmark [Params.PasswordHash] on representative
hardware and adjust via [WithTime], [WithMemory], and [WithThreads] so that
hashing takes 0.5–1 s under your expected load.

# Benefits

This package delivers a complete, OWASP-compliant password security layer in a
single import: correct algorithm, safe defaults, timing-attack-resistant
comparison, optional pepper support, and a portable self-describing storage
format — letting application code focus on business logic rather than
cryptographic plumbing.
*/
package passwordhash

import (
	"crypto/subtle"
	"errors"
	"fmt"

	"github.com/tecnickcom/gogen/pkg/encode"
	"github.com/tecnickcom/gogen/pkg/encrypt"
	"github.com/tecnickcom/gogen/pkg/random"
	"golang.org/x/crypto/argon2"
)

const (
	// DefaultAlgo is the default algorithm used to hash the password.
	// It corresponds to Type y=2.
	DefaultAlgo = "argon2id"

	// DefaultKeyLen is the default derived key length (Tag length) in bytes.
	// When configuring new hashes the key length is clamped to [16, 1024] bytes;
	// verification additionally accepts down to 4 bytes for legacy hashes.
	DefaultKeyLen = 32
	// minKeyLen is the smallest key length accepted when verifying a stored hash
	// (RFC 9106 validity bound). New hashes are held to the stricter minHashKeyLen.
	minKeyLen = 4
	// minHashKeyLen is the smallest key length allowed when configuring new hashes.
	// A 16-byte (128-bit) tag is the OWASP/RFC 9106 recommended floor; shorter keys
	// remain verifiable for backward compatibility but can no longer be minted.
	minHashKeyLen = 16

	// DefaultSaltLen is the default length of the random password salt (Nonce S).
	// When configuring new hashes the salt length is clamped to [8, 1024] bytes;
	// verification additionally accepts down to 1 byte for legacy hashes.
	// The value of 16 bytes is recommended for password hashing.
	DefaultSaltLen = 16
	// minSaltLen is the smallest salt length accepted when verifying a stored hash.
	// New hashes are held to the stricter minHashSaltLen.
	minSaltLen = 1
	// minHashSaltLen is the smallest salt length allowed when configuring new hashes.
	// An 8-byte (64-bit) salt is a conservative floor that keeps rainbow-table and
	// pre-computation attacks impractical; shorter salts remain verifiable.
	minHashSaltLen = 8

	// DefaultTime (t) is the default number of passes (iterations) over the memory.
	// It is clamped to [1, 1024] for both hashing and verification.
	DefaultTime = 3
	minTime     = 1

	// DefaultMemory is the default size of the memory in KiB.
	// It is clamped to [8, 4194304] KiB (up to 4 GiB) and, at hashing time,
	// rounded down to the nearest multiple of 4*p (with a floor of 8*p).
	DefaultMemory = 64 * 1024
	minMemory     = 8
	memBlock      = 4

	minThreads = 1
	// defaultThreads is the default parallelism (p); RFC 9106 §4 recommends p=4.
	// It is deliberately a flat constant rather than a value derived from
	// runtime.NumCPU(): Argon2 lanes are goroutines, so p=4 is valid on any host,
	// and a machine-independent default keeps the work factor reproducible across
	// heterogeneous fleets — otherwise hosts with different core counts would
	// mint hashes with different parameters and PasswordNeedsRehash would report
	// an upgrade on every alternating login, re-hashing forever.
	defaultThreads = 4

	// DefaultMinPasswordLength is the default minimum length of the input password (Message string P).
	// It must have a length not greater than 2^(32)-1 bytes.
	DefaultMinPasswordLength = 8
	minPasswordLength        = 1

	// DefaultMaxPasswordLength is the default maximum length of the input password (Message string P).
	// It must have a length not greater than 2^(32)-1 bytes.
	DefaultMaxPasswordLength = 4096

	// Valid AES pepper key lengths (in bytes) for the Encrypt* methods,
	// corresponding to AES-128, AES-192, and AES-256.
	aesKeyLen128 = 16
	aesKeyLen192 = 24
	aesKeyLen256 = 32

	// Maximum bounds accepted for parameters deserialized from a stored hash.
	// The verify path derives a key using parameters embedded in the hash blob,
	// so they are validated against these limits (and the min* constants above)
	// to avoid panics inside argon2 and unbounded memory allocation when the
	// blob is forged or corrupted.
	maxVerifyTime    = 1 << 10 // maximum number of passes (iterations) over the memory
	maxVerifyMemory  = 1 << 22 // maximum memory size in KiB (4 GiB)
	maxVerifyKeyLen  = 1 << 10 // maximum derived key length in bytes
	maxVerifySaltLen = 1 << 10 // maximum salt length in bytes

	// maxHashLen is the maximum accepted length in bytes of an encoded hash
	// string passed to the verification and rehash-check methods. The verify
	// bounds above (1024-byte key and salt) imply a maximum legitimate blob of
	// roughly 4 KiB after JSON and base64 encoding; 16 KiB leaves ample room for
	// format growth while preventing attacker-length inputs from being base64-
	// and JSON-decoded (or AES-GCM-opened) at all.
	maxHashLen = 16 << 10
)

// Sentinel errors returned by the package. Callers can match them with
// [errors.Is] to distinguish policy rejections from malformed input, algorithm
// mismatches (which signal a needed migration), and configuration mistakes.
var (
	// ErrPasswordTooShort is returned when a password is shorter than the
	// configured minimum length. It is enforced only when hashing (registration
	// policy), never when verifying an existing login.
	ErrPasswordTooShort = errors.New("the password is too short")

	// ErrPasswordTooLong is returned when a password exceeds the configured
	// maximum length. It guards both hashing and verification against
	// denial-of-service via extremely long inputs.
	ErrPasswordTooLong = errors.New("the password is too long")

	// ErrInvalidParams is returned by the hashing methods when the [Params]
	// configuration is incomplete or out of range (for example a zero-value
	// struct built without [New], or a field mutated to an invalid value).
	ErrInvalidParams = errors.New("invalid hashing parameters")

	// ErrInvalidHashData is returned when a stored hash blob cannot be decoded
	// (or decrypted), exceeds the maximum accepted size, is missing fields, or
	// embeds parameters that are out of range or internally inconsistent — all
	// signs of corruption or forgery.
	ErrInvalidHashData = errors.New("invalid hash data")

	// ErrAlgoMismatch is returned when a stored hash was produced by a different
	// algorithm than the verifying [Params] expects.
	ErrAlgoMismatch = errors.New("different algorithm type")

	// ErrVersionMismatch is returned when a stored hash was produced by a
	// different Argon2 version than the verifying [Params] expects.
	ErrVersionMismatch = errors.New("different argon2 versions")

	// ErrInvalidPepperKey is returned by the Encrypt* methods when the pepper key
	// is not a valid AES key length (16, 24, or 32 bytes).
	ErrInvalidPepperKey = errors.New("pepper key must be 16, 24, or 32 bytes")
)

// Params contains the Argon2id parameters and limits used for password hashing.
//
// A Params value returned by [New] is safe for concurrent use by multiple
// goroutines, provided its fields are not modified after construction. The
// exported fields exist so the configuration can be serialized alongside each
// hash; mutating them on a shared instance while hashing or verifying is a data
// race and may also invalidate the parameter invariants enforced by [New].
type Params struct {
	// Algo is the algorithm used to hash the password.
	// It corresponds to Type y=2.
	Algo string `json:"A"`

	// Version is the algorithm version.
	Version uint8 `json:"V"`

	// KeyLen is the length of the derived key (Tag length) in bytes.
	// New hashes use a value in [16, 1024]; verification accepts [4, 1024].
	KeyLen uint32 `json:"K"`

	// SaltLen is the length of the random password salt (Nonce S) in bytes.
	// New hashes use a value in [8, 1024]; verification accepts [1, 1024].
	// 16 bytes is recommended and the salt should be unique for each password.
	SaltLen uint32 `json:"S"`

	// Time (t) is the number of passes over the memory, a value in [1, 1024].
	Time uint32 `json:"T"`

	// Memory is the size of the memory in KiB, a value in [8, 4194304] (up to 4 GiB).
	// The actual number of blocks is m', which is m rounded down to the nearest multiple of 4*p.
	Memory uint32 `json:"M"`

	// Threads (p) is the degree of parallelism p that determines how many independent
	// (but synchronizing) computational chains (lanes) can be run.
	// According to RFC 9106 it must be an integer value from 1 to 2^(24)-1,
	// but in this implementation is limited to 2^(8)-1.
	Threads uint8 `json:"P"`

	// minPLen is the minimum length of the input password (Message string P).
	// It must have a length not greater than 2^(32)-1 bytes.
	minPLen uint32

	// maxPLen is the maximum length of the input password (Message string P).
	// It must have a length not greater than 2^(32)-1 bytes.
	maxPLen uint32

	// rnd is the random generator.
	rnd *random.Rnd
}

// Hashed stores a derived key together with the parameters needed to verify it.
type Hashed struct {
	// Params are the Argon2 parameters used to derive Key.
	Params *Params `json:"P"`

	// Salt is the password salt (Nonce S) of length Params.SaltLen.
	// The salt should be unique for each password.
	Salt []byte `json:"S"`

	// Key is the hashed password (Tag) of length Params.KeyLen.
	Key []byte `json:"K"`
}

// defaultParams returns the default parameter values.
func defaultParams() *Params {
	return &Params{
		Algo:    DefaultAlgo,
		Version: argon2.Version,
		KeyLen:  DefaultKeyLen,
		SaltLen: DefaultSaltLen,
		Time:    DefaultTime,
		Memory:  DefaultMemory,
		Threads: defaultThreads,
		minPLen: DefaultMinPasswordLength,
		maxPLen: DefaultMaxPasswordLength,
		rnd:     random.New(nil),
	}
}

// New creates a [Params] instance with OWASP-recommended Argon2id defaults,
// then applies any provided [Option] functions.
//
// Defaults:
//   - Algorithm:       argon2id (RFC 9106)
//   - KeyLen:          32 bytes
//   - SaltLen:         16 bytes
//   - Time:            3 passes
//   - Memory:          64 MiB
//   - Threads:         4 lanes (RFC 9106 §4), machine-independent by design
//   - MinPasswordLen:  8 bytes
//   - MaxPasswordLen:  4096 bytes
//
// Password lengths are measured in bytes, not Unicode characters: a multi-byte
// UTF-8 password counts its encoded length. Callers that need a character-based
// policy, or that must match a password typed on systems using different Unicode
// normalization forms (NFC vs NFD), should normalize (for example to NFKC) before
// hashing.
//
// Use [WithTime], [WithMemory], [WithThreads], [WithKeyLen], [WithSaltLen],
// [WithMinPasswordLength], and [WithMaxPasswordLength] to tune for your
// hardware profile before deploying to production.
func New(opts ...Option) *Params {
	ph := defaultParams()

	for _, applyOpt := range opts {
		applyOpt(ph)
	}

	ph.Memory = adjustMemory(ph.Memory, uint32(ph.Threads))

	// Cross-validate the password-length window. Done here rather than inside the
	// options so it is independent of the order options are applied: a max set
	// below the min (for example WithMaxPasswordLength(0)) would otherwise reject
	// every password.
	ph.maxPLen = max(ph.maxPLen, ph.minPLen)

	return ph
}

// PasswordHash hashes password using Argon2id and returns a portable,
// self-describing base64-encoded JSON string suitable for long-term storage.
//
// A cryptographically random salt of length [Params.SaltLen] is generated for
// each call, so two hashes of the same password will always differ.
// The returned string embeds the algorithm, version, all tuning parameters,
// the salt, and the derived key — everything needed for future verification.
//
// Returns an error if the configuration is invalid ([ErrInvalidParams], for
// example a zero-value Params built without [New]), if password is shorter
// than MinPasswordLength ([ErrPasswordTooShort]) or longer than
// MaxPasswordLength ([ErrPasswordTooLong]), or if random salt generation fails.
// Use [Params.PasswordVerify] to verify stored hashes.
func (ph *Params) PasswordHash(password string) (string, error) {
	data, err := ph.passwordHashData(password)
	if err != nil {
		return "", err
	}

	return encode.Serialize(data) //nolint:wrapcheck
}

// PasswordVerify checks whether password matches a hash produced by
// [Params.PasswordHash].
//
// The stored hash is decoded, its algorithm and version are validated, the
// candidate password is re-hashed with the stored parameters and salt, and
// the result is compared using [crypto/subtle.ConstantTimeCompare] to prevent
// timing attacks.
//
// The minimum-length policy is deliberately NOT enforced here: it is a
// registration-time policy applied by [Params.PasswordHash]. Enforcing it on
// verification would lock out existing users the moment the minimum is raised,
// returning an error instead of letting them authenticate and be migrated. The
// maximum-length guard is still applied, as it is the denial-of-service backstop.
//
// The bounds on parameters deserialized from the stored hash are intentionally
// generous backstops against corrupted or forged blobs: up to 1024 passes,
// 4 GiB of memory, and 1024-byte keys and salts. They assume stored hashes
// originate from a trusted store (your own database); a single forged blob at
// the upper bounds can still cost significant CPU and memory to verify. Hash
// strings longer than 16 KiB — four times the largest blob those bounds allow —
// are rejected before any decoding.
//
// Returns (true, nil) on a successful match, (false, nil) on a non-match, or
// (false, err) if password exceeds the configured maximum length, or if the
// hash string is oversized or malformed, embeds out-of-range parameters, or
// uses an incompatible algorithm/version (all matchable with [errors.Is]).
func (ph *Params) PasswordVerify(password, hash string) (bool, error) {
	err := validateHashLen(hash)
	if err != nil {
		return false, err
	}

	data := &Hashed{}

	err = encode.Deserialize(hash, data)
	if err != nil {
		return false, fmt.Errorf("%w: unable to decode the hash string: %w", ErrInvalidHashData, err)
	}

	return ph.passwordVerifyData(password, data)
}

// EncryptPasswordHash hashes password with Argon2id and then encrypts the
// resulting hash object with AES-GCM using key as the pepper.
//
// key is the AES pepper and must be a valid AES key length of 16, 24, or 32
// bytes (for AES-128, AES-192, or AES-256 respectively); any other length
// returns an error before hashing.
//
// Because the pepper is stored separately from the database (e.g. in a secrets
// manager or HSM), an attacker who obtains the database alone cannot perform
// an offline dictionary attack — the ciphertext is opaque without the key.
//
// The returned string is a base64-encoded AES-GCM ciphertext. Use
// [Params.EncryptPasswordVerify] with the same key to verify.
func (ph *Params) EncryptPasswordHash(key []byte, password string) (string, error) {
	err := validatePepperKey(key)
	if err != nil {
		return "", err
	}

	data, err := ph.passwordHashData(password)
	if err != nil {
		return "", err
	}

	return encrypt.EncryptSerializeAny(key, data) //nolint:wrapcheck
}

// EncryptPasswordVerify decrypts the hash produced by [Params.EncryptPasswordHash]
// using key, then verifies password against the decrypted Argon2id hash.
//
// key is the AES pepper and must be a valid AES key length of 16, 24, or 32
// bytes (for AES-128, AES-192, or AES-256 respectively); any other length
// returns an error before decryption.
//
// As with [Params.PasswordVerify], the minimum-length policy is not enforced on
// verification; the maximum-length guard is.
//
// Returns (true, nil) on a successful match, (false, nil) on a non-match, or
// (false, err) if password exceeds the configured maximum length, if decryption
// fails (wrong key, tampered ciphertext), or the inner hash is malformed.
func (ph *Params) EncryptPasswordVerify(key []byte, password, hash string) (bool, error) {
	err := validatePepperKey(key)
	if err != nil {
		return false, err
	}

	err = validateHashLen(hash)
	if err != nil {
		return false, err
	}

	data := &Hashed{}

	err = encrypt.DecryptSerializeAny(key, hash, data)
	if err != nil {
		return false, fmt.Errorf("%w: unable to decrypt the hash string: %w", ErrInvalidHashData, err)
	}

	ok, err := ph.passwordVerifyData(password, data)

	// Best-effort wipe of the decrypted material the pepper exists to keep out
	// of memory: without the AES key a database leak alone cannot expose it, so
	// avoid leaving a plaintext copy for the garbage collector.
	clear(data.Key)
	clear(data.Salt)

	return ok, err
}

// PasswordNeedsRehash reports whether a hash produced by [Params.PasswordHash]
// was created with parameters that differ from this [Params] configuration, and
// so should be re-hashed. Because the storage format is self-describing,
// parameters can be upgraded (or an algorithm/version changed) without
// invalidating stored hashes; PasswordNeedsRehash lets callers detect and
// transparently upgrade a stored hash on the next successful login.
//
// It returns true if the stored algorithm, version, key length, salt length,
// time, memory, or threads differ from the current configuration. It returns an
// error only if the hash string is oversized or malformed, or embeds
// out-of-range parameters; a well-formed hash that simply matches the current
// parameters returns (false, nil). PasswordNeedsRehash does not verify the
// password — call it after a successful [Params.PasswordVerify].
func (ph *Params) PasswordNeedsRehash(hash string) (bool, error) {
	err := validateHashLen(hash)
	if err != nil {
		return false, err
	}

	data := &Hashed{}

	err = encode.Deserialize(hash, data)
	if err != nil {
		return false, fmt.Errorf("%w: unable to decode the hash string: %w", ErrInvalidHashData, err)
	}

	return ph.needsRehashData(data)
}

// EncryptPasswordNeedsRehash is the pepper-encrypted counterpart to
// [Params.PasswordNeedsRehash] for hashes produced by
// [Params.EncryptPasswordHash]. key is the AES pepper and must be a valid AES
// key length (16, 24, or 32 bytes).
func (ph *Params) EncryptPasswordNeedsRehash(key []byte, hash string) (bool, error) {
	err := validatePepperKey(key)
	if err != nil {
		return false, err
	}

	err = validateHashLen(hash)
	if err != nil {
		return false, err
	}

	data := &Hashed{}

	err = encrypt.DecryptSerializeAny(key, hash, data)
	if err != nil {
		return false, fmt.Errorf("%w: unable to decrypt the hash string: %w", ErrInvalidHashData, err)
	}

	need, err := ph.needsRehashData(data)

	// Same best-effort wipe of decrypted material as in EncryptPasswordVerify.
	clear(data.Key)
	clear(data.Salt)

	return need, err
}

// validateHashLen rejects encoded hash strings longer than maxHashLen before
// any base64, JSON, or AES-GCM processing. The verify-side parameter bounds
// imply a maximum legitimate blob of about 4 KiB, so anything larger cannot be
// a valid hash and is refused without decoding it.
func validateHashLen(hash string) error {
	if len(hash) > maxHashLen {
		return fmt.Errorf("%w: hash string length %d exceeds %d bytes", ErrInvalidHashData, len(hash), maxHashLen)
	}

	return nil
}

// validatePepperKey checks that key is a valid AES key length (16, 24, or 32
// bytes) before it is passed to the AES-GCM encryption layer. Validating up
// front yields a clear, actionable error instead of a low-level cipher failure
// surfacing deep inside the encrypt package after the password has been hashed.
func validatePepperKey(key []byte) error {
	switch len(key) {
	case aesKeyLen128, aesKeyLen192, aesKeyLen256:
		return nil
	default:
		return fmt.Errorf("%w, got %d", ErrInvalidPepperKey, len(key))
	}
}

// passwordHashData generates a hashed password using the provided password string.
// It generates a random salt of length ph.SaltLen and uses the argon2id algorithm
// to hash the password with the salt, using the parameters specified in ph.
// The resulting hashed password, the salt and the parameters are returned as a struct.
func (ph *Params) passwordHashData(password string) (*Hashed, error) {
	err := ph.validateHashParams()
	if err != nil {
		return nil, err
	}

	if len(password) < int(ph.minPLen) {
		return nil, fmt.Errorf("%w: %d < %d", ErrPasswordTooShort, len(password), ph.minPLen)
	}

	if len(password) > int(ph.maxPLen) {
		return nil, fmt.Errorf("%w: %d > %d", ErrPasswordTooLong, len(password), ph.maxPLen)
	}

	salt, err := ph.rnd.RandomBytes(int(ph.SaltLen))
	if err != nil {
		return nil, err //nolint:wrapcheck
	}

	return &Hashed{
		Params: &Params{
			Algo:    ph.Algo,
			Version: ph.Version,
			KeyLen:  ph.KeyLen,
			SaltLen: ph.SaltLen,
			Time:    ph.Time,
			Memory:  ph.Memory,
			Threads: ph.Threads,
		},
		Salt: salt,
		Key:  argon2.IDKey([]byte(password), salt, ph.Time, ph.Memory, ph.Threads, ph.KeyLen),
	}, nil
}

// passwordVerifyData verifies if a given password matches a hashed password generated with the passwordHashData method.
// It returns true if the password matches the hashed password, otherwise false.
// The maximum password length and the deserialized parameters are validated
// before any CPU-intensive computation takes place.
func (ph *Params) passwordVerifyData(password string, data *Hashed) (bool, error) {
	// maxPLen is unexported and always >= 1 after New, so zero identifies a
	// receiver built without New; fail with a clear configuration error instead
	// of the misleading "password too long: N > 0".
	if ph.maxPLen == 0 {
		return false, fmt.Errorf("%w: uninitialized parameters (use New)", ErrInvalidParams)
	}

	// The minimum-length policy is intentionally not enforced on verification:
	// it is a registration-time policy, and enforcing it here would lock out
	// users whose stored password predates a raised minimum. The maximum-length
	// guard stays, as it bounds the cost of the Argon2 computation below.
	if len(password) > int(ph.maxPLen) {
		return false, fmt.Errorf("%w: %d > %d", ErrPasswordTooLong, len(password), ph.maxPLen)
	}

	err := validateVerifyData(data)
	if err != nil {
		return false, err
	}

	if data.Params.Algo != ph.Algo {
		return false, fmt.Errorf("%w: lib=%s, hash=%s", ErrAlgoMismatch, ph.Algo, data.Params.Algo)
	}

	if data.Params.Version != ph.Version {
		return false, fmt.Errorf("%w: lib=%d, hash=%d", ErrVersionMismatch, ph.Version, data.Params.Version)
	}

	newkey := argon2.IDKey([]byte(password), data.Salt, data.Params.Time, data.Params.Memory, data.Params.Threads, data.Params.KeyLen)

	match := subtle.ConstantTimeCompare(newkey, data.Key) == 1

	// Best-effort wipe of the freshly derived key. Go cannot guarantee no copies
	// remain (stack growth, GC), so this is defense in depth, not a guarantee.
	clear(newkey)

	return match, nil
}

// needsRehashData reports whether the parameters embedded in data differ from
// the current configuration, after validating that they are well-formed.
func (ph *Params) needsRehashData(data *Hashed) (bool, error) {
	// Same uninitialized-receiver guard as passwordVerifyData: comparing stored
	// parameters against a zero-value configuration would answer "needs rehash"
	// for garbage reasons instead of flagging the misuse.
	if ph.maxPLen == 0 {
		return false, fmt.Errorf("%w: uninitialized parameters (use New)", ErrInvalidParams)
	}

	err := validateVerifyData(data)
	if err != nil {
		return false, err
	}

	p := data.Params

	changed := p.Algo != ph.Algo ||
		p.Version != ph.Version ||
		p.KeyLen != ph.KeyLen ||
		p.SaltLen != ph.SaltLen ||
		p.Time != ph.Time ||
		p.Memory != ph.Memory ||
		p.Threads != ph.Threads

	return changed, nil
}

// validateHashParams checks that the receiver is a complete, in-range
// configuration before it is used to hash a password. It converts what would
// otherwise be a panic inside argon2 (zero time, threads, or key length) or a
// nil-pointer dereference (missing random source) into a returned error, so a
// zero-value or mutated [Params] can never crash the caller. It also rejects
// any Algo or Version the implementation cannot produce: hashing always uses
// argon2id at [argon2.Version], so stamping a different label into the
// self-describing blob would be a lie discovered only at first verification
// (by this package) or by a cross-language consumer trusting the label.
func (ph *Params) validateHashParams() error {
	if ph.rnd == nil {
		return fmt.Errorf("%w: missing random source (use New)", ErrInvalidParams)
	}

	if ph.maxPLen < ph.minPLen {
		return fmt.Errorf("%w: max password length %d < min %d", ErrInvalidParams, ph.maxPLen, ph.minPLen)
	}

	if ph.Algo != DefaultAlgo {
		return fmt.Errorf("%w: algorithm=%q (only %q can be produced)", ErrInvalidParams, ph.Algo, DefaultAlgo)
	}

	if ph.Version != argon2.Version {
		return fmt.Errorf("%w: version=%d (only %d can be produced)", ErrInvalidParams, ph.Version, argon2.Version)
	}

	return ph.validateHashCostParams()
}

// validateHashCostParams checks the Argon2 cost parameters against the mint
// envelope: the same [floor, ceiling] range the verification path accepts, so
// that any hash this configuration can mint, it can also verify. The ceilings
// matter as much as the floors — a value above the verify limit (for example
// Time > 1024 or Memory > 4 GiB) would mint a blob that every verification path
// rejects, a total lockout discovered only at first login. The KeyLen and
// SaltLen floors are the stricter minHashKeyLen and minHashSaltLen (not the
// looser verify-time minimums), so sub-floor hashes cannot be minted even by
// mutating the exported fields directly; hashes that already exist outside the
// mint envelope remain verifiable.
func (ph *Params) validateHashCostParams() error {
	if outOfRange(uint64(ph.Time), minTime, maxVerifyTime) {
		return fmt.Errorf("%w: time=%d (allowed %d..%d)", ErrInvalidParams, ph.Time, minTime, maxVerifyTime)
	}

	if ph.Threads < minThreads {
		return fmt.Errorf("%w: threads=%d (minimum %d)", ErrInvalidParams, ph.Threads, minThreads)
	}

	if outOfRange(uint64(ph.KeyLen), minHashKeyLen, maxVerifyKeyLen) {
		return fmt.Errorf("%w: key length=%d (allowed %d..%d)", ErrInvalidParams, ph.KeyLen, minHashKeyLen, maxVerifyKeyLen)
	}

	if outOfRange(uint64(ph.SaltLen), minHashSaltLen, maxVerifySaltLen) {
		return fmt.Errorf("%w: salt length=%d (allowed %d..%d)", ErrInvalidParams, ph.SaltLen, minHashSaltLen, maxVerifySaltLen)
	}

	if outOfRange(uint64(ph.Memory), minMemory, maxVerifyMemory) {
		return fmt.Errorf("%w: memory=%d (allowed %d..%d)", ErrInvalidParams, ph.Memory, minMemory, maxVerifyMemory)
	}

	return nil
}

// validateVerifyData checks that the parameters deserialized from a stored
// hash are present and within sane bounds before they are fed to argon2.IDKey.
// It returns an error for nil or out-of-range values instead of letting argon2
// panic (or allocate unbounded memory) on forged or corrupted hash blobs.
func validateVerifyData(data *Hashed) error {
	if data == nil || data.Params == nil {
		return fmt.Errorf("%w: missing hash parameters", ErrInvalidHashData)
	}

	err := validateStoredParamBounds(data.Params)
	if err != nil {
		return err
	}

	return validateStoredLengths(data)
}

// validateStoredParamBounds checks each numeric Argon2 parameter deserialized
// from a stored hash against its accepted [min, max] verification bound.
func validateStoredParamBounds(p *Params) error {
	if outOfRange(uint64(p.Time), minTime, maxVerifyTime) {
		return fmt.Errorf("%w: invalid time parameter: %d", ErrInvalidHashData, p.Time)
	}

	if outOfRange(uint64(p.Memory), minMemory, maxVerifyMemory) {
		return fmt.Errorf("%w: invalid memory parameter: %d", ErrInvalidHashData, p.Memory)
	}

	if p.Threads < minThreads {
		return fmt.Errorf("%w: invalid threads parameter: %d", ErrInvalidHashData, p.Threads)
	}

	if outOfRange(uint64(p.KeyLen), minKeyLen, maxVerifyKeyLen) {
		return fmt.Errorf("%w: invalid key length parameter: %d", ErrInvalidHashData, p.KeyLen)
	}

	if outOfRange(uint64(p.SaltLen), minSaltLen, maxVerifySaltLen) {
		return fmt.Errorf("%w: invalid salt length parameter: %d", ErrInvalidHashData, p.SaltLen)
	}

	return nil
}

// validateStoredLengths checks that the actual salt and key byte lengths match
// the lengths declared in the stored parameters. A blob whose Key or Salt does
// not match its declared length is internally inconsistent (corrupt or forged);
// rejecting it up front avoids silently treating a corrupt record as a simple
// password mismatch. Bounds on the declared lengths are enforced by
// [validateStoredParamBounds], so the equality below also bounds the actuals.
func validateStoredLengths(data *Hashed) error {
	if len(data.Key) != int(data.Params.KeyLen) {
		return fmt.Errorf("%w: key length %d does not match declared %d", ErrInvalidHashData, len(data.Key), data.Params.KeyLen)
	}

	if len(data.Salt) != int(data.Params.SaltLen) {
		return fmt.Errorf("%w: salt length %d does not match declared %d", ErrInvalidHashData, len(data.Salt), data.Params.SaltLen)
	}

	return nil
}

// outOfRange reports whether value falls outside the [minValue, maxValue] interval.
func outOfRange(value, minValue, maxValue uint64) bool {
	return (value < minValue) || (value > maxValue)
}

// adjustMemory returns the actual number of blocks is m',
// which is m rounded down to the nearest multiple of 4*p.
func adjustMemory(m uint32, p uint32) uint32 {
	block := (memBlock * p)
	return max((2 * block), ((m / block) * block))
}
