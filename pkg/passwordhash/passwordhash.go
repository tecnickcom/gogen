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
recommendations into two method pairs on a single [Params] configuration object:

	p := passwordhash.New() // sensible OWASP defaults

	// Hash a password for storage.
	hash, err := p.PasswordHash(plaintext)

	// Verify a login attempt.
	ok, err := p.PasswordVerify(plaintext, hash)

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
	    "P": 16           // parallelism (threads)
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
  - Input length guards: passwords are rejected before any CPU-intensive
    computation if they fall outside the configured min/max length, preventing
    denial-of-service via extremely long inputs.
  - Tunable via functional options: [WithTime], [WithMemory], [WithThreads],
    [WithKeyLen], [WithSaltLen], [WithMinPasswordLength], and
    [WithMaxPasswordLength] let each deployment match the OWASP parameter
    guidance for its hardware profile.
  - Portable format: JSON + base64 storage can be decoded by any language with
    standard libraries, enabling cross-platform password verification.

# Verification Flow

 1. Decode the stored base64 string to retrieve the JSON object.
 2. Unmarshal the JSON to recover algorithm, version, parameters, and salt.
 3. Validate that the stored algorithm and version match the library.
 4. Re-derive the key from the candidate password using the stored parameters
    and salt.
 5. Compare the derived key against the stored key with [crypto/subtle.ConstantTimeCompare]
    to prevent timing attacks.

# Parameter Tuning

The defaults (T=3, M=64 MiB, P=NumCPU) follow OWASP and RFC 9106 §4
(https://datatracker.ietf.org/doc/html/rfc9106#section-4) recommendations.
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
	"fmt"
	"runtime"

	"github.com/tecnickcom/gogen/pkg/encode"
	"github.com/tecnickcom/gogen/pkg/encrypt"
	"github.com/tecnickcom/gogen/pkg/random"
	"golang.org/x/crypto/argon2"
)

const (
	// DefaultAlgo is the default algorithm used to hash the password.
	// It corresponds to Type y=2.
	DefaultAlgo = "argon2id"

	// DefaultKeyLen is the default length of the returned byte-slice that can be used as cryptographic key (Tag length).
	// It must be an integer number of bytes from 4 to 2^(32)-1.
	DefaultKeyLen = 32
	minKeyLen     = 4

	// DefaultSaltLen is the default length of the random password salt (Nonce S).
	// It must be not greater than 2^(32)-1 bytes.
	// The value of 16 bytes is recommended for password hashing.
	DefaultSaltLen = 16
	minSaltLen     = 1

	// DefaultTime (t) is the default number of passes (iterations) over the memory.
	// It must be an integer value from 1 to 2^(32)-1.
	DefaultTime = 3
	minTime     = 1

	// DefaultMemory is the default size of the memory in KiB.
	// It must be an integer number of kibibytes from 8*p to 2^(32)-1.
	// The actual number of blocks is m', which is m rounded down to the nearest multiple of 4*p.
	DefaultMemory = 64 * 1024
	minMemory     = 8
	memBlock      = 4

	minThreads = 1
	maxThreads = 255

	// DefaultMinPasswordLength is the default minimum length of the input password (Message string P).
	// It must have a length not greater than 2^(32)-1 bytes.
	DefaultMinPasswordLength = 8

	// DefaultMaxPasswordLength is the default maximum length of the input password (Message string P).
	// It must have a length not greater than 2^(32)-1 bytes.
	DefaultMaxPasswordLength = 4096
)

// Params contains the Argon2id parameters and limits used for password hashing.
type Params struct {
	// Algo is the algorithm used to hash the password.
	// It corresponds to Type y=2.
	Algo string `json:"A"`

	// Version is the algorithm version.
	Version uint8 `json:"V"`

	// KeyLen is the length of the returned byte-slice that can be used as cryptographic key (Tag length).
	// It must be an integer number of bytes from 4 to 2^(32)-1.
	KeyLen uint32 `json:"K"`

	// SaltLen is the length of the random password salt (Nonce S).
	// It must be not greater than 2^(32)-1 bytes.
	// The value of 16 bytes is recommended for password hashing.
	SaltLen uint32 `json:"S"`

	// Time (t) is the default number of passes over the memory.
	// It must be an integer value from 1 to 2^(32)-1.
	Time uint32 `json:"T"`

	// Memory is the size of the memory in KiB.
	// It must be an integer number of kibibytes from 8*p to 2^(32)-1.
	// The actual number of blocks is m', which is m rounded down to the nearest multiple of 4*p.
	Memory uint32 `json:"M"`

	// Threads (p) is the degree of parallelism p that determines how many independent
	// (but	synchronizing) computational chains (lanes) can be run.
	// According to the RFC9106 it must be an integer value from 1 to 2^(24)-1,
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
		Threads: uint8(max(minThreads, min(runtime.NumCPU(), maxThreads))),
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
//   - Threads:         runtime.NumCPU() clamped to [1, 255]
//   - MinPasswordLen:  8 characters
//   - MaxPasswordLen:  4096 characters
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
// Returns an error if password is shorter than MinPasswordLength or longer
// than MaxPasswordLength, or if random salt generation fails.
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
// Returns (true, nil) on a successful match, (false, nil) on a non-match, or
// (false, err) if the hash string is malformed or uses an incompatible
// algorithm/version.
func (ph *Params) PasswordVerify(password, hash string) (bool, error) {
	data := &Hashed{}

	err := encode.Deserialize(hash, data)
	if err != nil {
		return false, fmt.Errorf("unable to decode the hash string: %w", err)
	}

	return ph.passwordVerifyData(password, data)
}

// EncryptPasswordHash hashes password with Argon2id and then encrypts the
// resulting hash object with AES-GCM using key as the pepper.
//
// Because the pepper is stored separately from the database (e.g. in a secrets
// manager or HSM), an attacker who obtains the database alone cannot perform
// an offline dictionary attack — the ciphertext is opaque without the key.
//
// The returned string is a base64-encoded AES-GCM ciphertext. Use
// [Params.EncryptPasswordVerify] with the same key to verify.
func (ph *Params) EncryptPasswordHash(key []byte, password string) (string, error) {
	data, err := ph.passwordHashData(password)
	if err != nil {
		return "", err
	}

	return encrypt.EncryptSerializeAny(key, data) //nolint:wrapcheck
}

// EncryptPasswordVerify decrypts the hash produced by [Params.EncryptPasswordHash]
// using key, then verifies password against the decrypted Argon2id hash.
//
// Returns (true, nil) on a successful match, (false, nil) on a non-match, or
// (false, err) if decryption fails (wrong key, tampered ciphertext) or the
// inner hash is malformed.
func (ph *Params) EncryptPasswordVerify(key []byte, password, hash string) (bool, error) {
	data := &Hashed{}

	err := encrypt.DecryptSerializeAny(key, hash, data)
	if err != nil {
		return false, fmt.Errorf("unable to decode the hash string: %w", err)
	}

	return ph.passwordVerifyData(password, data)
}

// passwordHashData generates a hashed password using the provided password string.
// It generates a random salt of length ph.SaltLen and uses the argon2id algorithm
// to hash the password with the salt, using the parameters specified in ph.
// The resulting hashed password, the salt and the parameters are returned as a struct.
func (ph *Params) passwordHashData(password string) (*Hashed, error) {
	if len(password) < int(ph.minPLen) {
		return nil, fmt.Errorf("the password is too short: %d > %d", len(password), ph.minPLen)
	}

	if len(password) > int(ph.maxPLen) {
		return nil, fmt.Errorf("the password is too long: %d > %d", len(password), ph.maxPLen)
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
func (ph *Params) passwordVerifyData(password string, data *Hashed) (bool, error) {
	if data.Params.Algo != ph.Algo {
		return false, fmt.Errorf("different algorithm type: lib=%s, hash=%s", ph.Algo, data.Params.Algo)
	}

	if data.Params.Version != ph.Version {
		return false, fmt.Errorf("different argon2 versions: lib=%d, hash=%d", ph.Version, data.Params.Version)
	}

	newkey := argon2.IDKey([]byte(password), data.Salt, data.Params.Time, data.Params.Memory, data.Params.Threads, data.Params.KeyLen)

	return subtle.ConstantTimeCompare(newkey, data.Key) == 1, nil
}

// adjustMemory returns the actual number of blocks is m',
// which is m rounded down to the nearest multiple of 4*p.
func adjustMemory(m uint32, p uint32) uint32 {
	block := (memBlock * p)
	return max((2 * block), ((m / block) * block))
}
