/*
Package passwordhash contains functions to create and verify a password hash using a strong one-way hashing algorithm.

It supports "peppering" by encrypting the hashed passwords using a secret key.

It is based on the Argon2id algorithm as recommended by the OWASP Password Storage Cheat Sheet:
https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html
*/
package passwordhash

import (
	"crypto/subtle"
	"fmt"
	"runtime"

	"github.com/Vonage/gosrvlib/pkg/typeutil"
	"golang.org/x/crypto/argon2"
)

const (
	// DefaultAlgo is the default algorithm used to hash the password.
	DefaultAlgo = "argon2id"

	// DefaultKeyLen is the default length of the returned byte-slice that can be used as cryptographic key.
	DefaultKeyLen = 32

	// DefaultSaltLen is the default length of the random password salt.
	DefaultSaltLen = 16

	// DefaultTime is the default number of passes over the memory.
	DefaultTime = 1

	// DefaultMemory is the default size of the memory in KiB.
	DefaultMemory = 65_536
)

// Params contains the parameters for hashing the password.
type Params struct {
	// Algo is the algorithm used to hash the password.
	Algo string `json:"A"`

	// Version is the algorithm version implemented by the parent package.
	Version uint8 `json:"V"`

	// KeyLen is the length of the returned byte-slice that can be used as cryptographic key.
	KeyLen uint32 `json:"K"`

	// SaltLen is the length of the random password salt.
	SaltLen uint32 `json:"S"`

	// Time is the number of passes over the memory.
	Time uint32 `json:"T"`

	// Memory is the size of the memory in KiB.
	Memory uint32 `json:"M"`

	// Threads is number of threads used by hashing the algorithm.
	Threads uint8 `json:"P"`
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
		Threads: uint8(runtime.NumCPU()),
	}
}

// New creates a new instance of Params with the provided options applied.
func New(opts ...Option) *Params {
	ph := defaultParams()

	for _, applyOpt := range opts {
		applyOpt(ph)
	}

	return ph
}

// Hashed contains the hashed password key and hashing parameters.
type Hashed struct {
	// Params contains the hashing parameters.
	Params *Params `json:"P"`

	// Salt is the password salt.
	Salt []byte `json:"S"`

	// Key is the hashed password.
	Key []byte `json:"K"`
}

func (ph *Params) passwordHashData(password string) (*Hashed, error) {
	salt, err := typeutil.RandomBytes(typeutil.RandReader, int(ph.SaltLen))
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

	return typeutil.Serialize(data) //nolint:wrapcheck
}

// PasswordVerify verifies if a given password matches a hashed password generated with the PasswordHash method.
// It returns true if the password matches the hashed password, otherwise false.
func (ph *Params) PasswordVerify(password, hash string) (bool, error) {
	data := &Hashed{}

	err := typeutil.Deserialize(hash, data)
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

	return typeutil.EncryptSerializeAny(key, data) //nolint:wrapcheck
}

// EncryptPasswordVerify extends the PasswordVerify method by decrypting the password hash using the provided key (pepper).
func (ph *Params) EncryptPasswordVerify(key []byte, password, hash string) (bool, error) {
	data := &Hashed{}

	err := typeutil.DecryptSerializeAny(key, hash, data)
	if err != nil {
		return false, fmt.Errorf("unable to decode the hash string: %w", err)
	}

	return ph.passwordVerifyData(password, data)
}
