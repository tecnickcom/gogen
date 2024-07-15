/*
Package passwordhash implements a practical model to create and verify a
password hashes using a strong one-way hashing algorithm.

The model implements the best advice of OWASP Password Storage Cheat
(https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html)
with specific default settings (parameters) and encoding format. Is uses the
Argon2id algorithm (https://www.rfc-editor.org/info/rfc9106) as implemented in
the golang.org/x/crypto/argon2 package.

The implementation is based on available standard cryptographic libraries,
enforcing standard settings, and the way parameters and salt are encoded
alongside the hashed password.

Password Storage Object

  - The input password is checked for min/max length. This prevents any
    computation if these requirements are not met.

  - The input password is hashed using the Argon2id algorithm with some default
    parameters and a random salt. Argon2id is currently the best recommendation
    from OWASP and it is a variant of Argon2 that provides a balanced approach
    to resisting both side-channel (Argon2i) and GPU-based (Argon2d) attacks.

  - The hashed password is then encoded as base64 and added as a "K" field in a
    JSON object, alongside the Argon2id parameters and base64 encoded salt.

  - The JSON object is then encoded as a base64 string ready for storage.

Example:

	Password: "Test-Password-01234"

	{
	"P": { < Argon2id parameters.
	"A": "argon2id", < Name of the hashing algorithm (always "argon2id").
	"V": 19,         < Argon2id algorithm version (0x13).
	"K": 32,         < Length of the returned byte-slice that can be used as a cryptographic key.
	"S": 16,         < Length of the random password salt.
	"T": 3,          < Number of passes over the memory.
	"M": 65536,      < Size of the memory in KiB.
	"P": 16          < Number of threads used by the hashing algorithm.
	},
	"S": "wQYm4bfktbHq2omIwFu+4Q==", < base64-encoded random salt of "P.S" length.
	"K": "aU8hO900Odq6aKtWiWz3RW9ygn734liJaPtM6ynvkYI=" < base64-encoded Argon2id password hash.
	}

While other serialization methods are available, JSON and base64 have been
chosen for their extraordinary portability and compatibility with multiple
systems and programming languages. The proposed JSON schema is such that it can
be easily adapted to other hashing algorithms if required.

NOTE: Custom parameters should be agreed for production following the
recommendations at:
https://datatracker.ietf.org/doc/html/draft-irtf-cfrg-argon2-04#section-4 The
current reference implementation uses recommended and sensible values by
default.

The JSON object is then encoded as base64 string for storage (200 bytes in this
example):

	eyJQIjp7IkEiOiJhcmdvbjJpZCIsIlYiOjE5LCJLIjozMiwiUyI6MTYsIlQiOjMsIk0iOjY1NTM2LCJQIjoxNn0sIlMiOiJ3UVltNGJma3RiSHEyb21Jd0Z1KzRRPT0iLCJLIjoiYVU4aE85MDBPZHE2YUt0V2lXejNSVzl5Z243MzRsaUphUHRNNnludmtZST0ifQo=

Password Verification

  - The hashed password object string is retrieved from the storage and it is
    decoded using base64.

  - The resulting JSON is also decoded (unmarshalled) to access the fields
    values.

  - The field P.A (name of the hashing algorithm) is compared with the one in
    the library to ensure we are using the correct algorithm.

  - The field P.V (algorithm version) is compared with the one in the library to
    ensure we are using the correct version.

  - The provided live password is hashed using the same Argon2id algorithm with
    the parameters extracted from the JSON.

  - The hash of the live password is compared with the one retrieved from the
    JSON P.K field. The time taken for the comparison is a function of the
    length of the slices and is independent of the contents. This prevents
    timing attacks.
*/
package passwordhash

import (
	"crypto/subtle"
	"fmt"
	"runtime"

	"github.com/Vonage/gosrvlib/pkg/encode"
	"github.com/Vonage/gosrvlib/pkg/encrypt"
	"github.com/Vonage/gosrvlib/pkg/random"
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

// Params contains the parameters for hashing the password.
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

// Hashed contains the hashed password key and hashing parameters.
type Hashed struct {
	// Params contains the hashing parameters.
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

// New creates a new instance of Params with the provided options applied.
func New(opts ...Option) *Params {
	ph := defaultParams()

	for _, applyOpt := range opts {
		applyOpt(ph)
	}

	ph.Memory = adjustMemory(ph.Memory, uint32(ph.Threads))

	return ph
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

// PasswordHash generates a hashed password using the provided password string.
// It generates a random salt of length ph.SaltLen and uses the argon2id algorithm
// to hash the password with the salt, using the parameters specified in ph.
// The resulting hashed password, the salt and the parameters are returned as a json encoded as a base64 string.
func (ph *Params) PasswordHash(password string) (string, error) {
	data, err := ph.passwordHashData(password)
	if err != nil {
		return "", err
	}

	return encode.Serialize(data) //nolint:wrapcheck
}

// PasswordVerify verifies if a given password matches a hashed password generated with the PasswordHash method.
// It returns true if the password matches the hashed password, otherwise false.
func (ph *Params) PasswordVerify(password, hash string) (bool, error) {
	data := &Hashed{}

	err := encode.Deserialize(hash, data)
	if err != nil {
		return false, fmt.Errorf("unable to decode the hash string: %w", err)
	}

	return ph.passwordVerifyData(password, data)
}

// EncryptPasswordHash extends the PasswordHash method by encrypting the password hash using the provided key (pepper).
// As the key is not stored along the password hash, it provides an additional layer of protection.
func (ph *Params) EncryptPasswordHash(key []byte, password string) (string, error) {
	data, err := ph.passwordHashData(password)
	if err != nil {
		return "", err
	}

	return encrypt.EncryptSerializeAny(key, data) //nolint:wrapcheck
}

// EncryptPasswordVerify extends the PasswordVerify method by decrypting the password hash using the provided key (pepper).
func (ph *Params) EncryptPasswordVerify(key []byte, password, hash string) (bool, error) {
	data := &Hashed{}

	err := encrypt.DecryptSerializeAny(key, hash, data)
	if err != nil {
		return false, fmt.Errorf("unable to decode the hash string: %w", err)
	}

	return ph.passwordVerifyData(password, data)
}

// adjustMemory returns the actual number of blocks is m',
// which is m rounded down to the nearest multiple of 4*p.
func adjustMemory(m uint32, p uint32) uint32 {
	block := (memBlock * p)
	return max((2 * block), ((m / block) * block))
}
