package passwordhash

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/stretchr/testify/require"
	"github.com/tecnickcom/gogen/pkg/encode"
	"github.com/tecnickcom/gogen/pkg/random"
	"golang.org/x/crypto/argon2"
)

// testRefHash is a frozen wire-format reference: a hash of the 4-byte password
// "test" minted by an older build with legacy parameters (T=1, M=65536, P=16).
// It pins backward compatibility of the storage format: it must keep verifying
// and must keep reporting that a rehash is needed under current defaults.
const testRefHash = "eyJQIjp7IkEiOiJhcmdvbjJpZCIsIlYiOjE5LCJLIjozMiwiUyI6MTYsIlQiOjEsIk0iOjY1NTM2LCJQIjoxNn0sIlMiOiI1d25uaXRVaGV6cjFnbkdoeU1FVTdBPT0iLCJLIjoiQmNiUlRVNFNDcmQxNGJWUzRzcVBGYndvbnYreWlvZ09ueGJWMXBRTGRWMD0ifQo="

func TestNew(t *testing.T) {
	t.Parallel()

	p := New()

	require.NotEmpty(t, p.Algo)
	require.NotZero(t, p.Version)
	// The default parallelism is a flat constant, never derived from the number
	// of CPU cores: a machine-dependent default would make PasswordNeedsRehash
	// ping-pong between hosts with different core counts.
	require.Equal(t, uint8(defaultThreads), p.Threads)

	opts := []Option{
		WithKeyLen(31),
		WithSaltLen(17),
		WithTime(3),
		WithMemory(65_537),
		WithThreads(5),
		WithMinPasswordLength(16),
		WithMaxPasswordLength(128),
	}

	p = New(opts...)

	require.Equal(t, DefaultAlgo, p.Algo)
	require.Equal(t, uint8(argon2.Version), p.Version)
	require.Equal(t, uint32(31), p.KeyLen)
	require.Equal(t, uint32(17), p.SaltLen)
	require.Equal(t, uint32(3), p.Time)
	require.Equal(t, uint32(0xfff0), p.Memory)
	require.Equal(t, uint8(5), p.Threads)
	require.Equal(t, uint32(16), p.minPLen)
	require.Equal(t, uint32(128), p.maxPLen)
}

func Test_passwordHashData(t *testing.T) {
	t.Parallel()

	p := New()

	hash, err := p.passwordHashData("test-password")

	require.NoError(t, err)
	require.NotEmpty(t, hash)

	shortPassword := string(make([]byte, p.minPLen-1))

	hash, err = p.passwordHashData(shortPassword)

	require.Error(t, err)
	require.Empty(t, hash)
	require.Contains(t, err.Error(), fmt.Sprintf("the password is too short: %d < %d", p.minPLen-1, p.minPLen))

	longPassword := string(make([]byte, p.maxPLen+1))

	hash, err = p.passwordHashData(longPassword)

	require.Error(t, err)
	require.Empty(t, hash)

	// The password must satisfy the length policy so the failure below can only
	// come from the salt generator; a short password would fail the length check
	// first and the error reader would never be reached.
	p.rnd = random.New(iotest.ErrReader(errors.New("test-rand-reader-error")))

	hash, err = p.passwordHashData("test-password")

	require.Error(t, err)
	require.ErrorContains(t, err, "test-rand-reader-error")
	require.Empty(t, hash)
}

func Test_passwordHashData_passwordVerifyData(t *testing.T) {
	t.Parallel()

	p := New()

	secret := "test-secret-string"
	data, err := p.passwordHashData(secret)

	require.NoError(t, err)
	require.NotEmpty(t, data)

	ok, err := p.passwordVerifyData(secret, data)

	require.NoError(t, err)
	require.True(t, ok)

	ok, err = p.passwordVerifyData("test-wrong-secret", data)

	require.NoError(t, err)
	require.False(t, ok)

	p.Algo = "wrong-algo"

	ok, err = p.passwordVerifyData(secret, data)

	require.Error(t, err)
	require.False(t, ok)

	p.Algo = DefaultAlgo
	p.Version = 0

	ok, err = p.passwordVerifyData(secret, data)

	require.Error(t, err)
	require.False(t, ok)
}

func TestPasswordHash(t *testing.T) {
	t.Parallel()

	p := New()

	hash, err := p.PasswordHash("TestPasswordString")

	require.NoError(t, err)
	require.NotEmpty(t, hash)

	// Long enough to pass the length policy: the failure must come from the
	// salt generator, not from an earlier guard.
	p.rnd = random.New(iotest.ErrReader(errors.New("test-rand-reader-error")))

	_, err = p.PasswordHash("TestPasswordString")

	require.Error(t, err)
	require.ErrorContains(t, err, "test-rand-reader-error")
}

func TestPasswordVerify(t *testing.T) {
	t.Parallel()

	// The reference hash was generated for the 4-character password "test".
	// Verification does not enforce the minimum-length policy, so a default
	// configuration (min length 8) still verifies this legacy short password.
	p := New()

	ok, err := p.PasswordVerify("test", testRefHash)

	require.NoError(t, err)
	require.True(t, ok)

	ok, err = p.PasswordVerify("secret", "wrong-hash")

	require.Error(t, err)
	require.ErrorIs(t, err, ErrInvalidHashData)
	require.False(t, ok)
}

func TestPasswordVerifyLengthGuards(t *testing.T) {
	t.Parallel()

	p := New()

	secret := "Test-Password-01234"

	hash, err := p.PasswordHash(secret)

	require.NoError(t, err)
	require.NotEmpty(t, hash)

	// Shorter than MinPasswordLength: verification does NOT enforce the minimum
	// (registration-time policy), so a short candidate is treated as an ordinary
	// non-match rather than an error. This prevents locking out existing users
	// when the minimum length is later raised.
	ok, err := p.PasswordVerify("short", hash)

	require.NoError(t, err)
	require.False(t, ok)

	// Longer than MaxPasswordLength: rejected before any Argon2 computation.
	ok, err = p.PasswordVerify(strings.Repeat("x", int(p.maxPLen)+1), hash)

	require.Error(t, err)
	require.ErrorIs(t, err, ErrPasswordTooLong)
	require.False(t, ok)
	require.Contains(t, err.Error(), "the password is too long")
}

func TestPasswordVerifyInvalidHashParams(t *testing.T) {
	t.Parallel()

	p := New()

	validParams := func() *Params {
		return &Params{
			Algo:    DefaultAlgo,
			Version: argon2.Version,
			KeyLen:  DefaultKeyLen,
			SaltLen: DefaultSaltLen,
			Time:    1,
			Memory:  DefaultMemory,
			Threads: 1,
		}
	}

	tests := []struct {
		name string
		data *Hashed
	}{
		{
			name: "missing params",
			data: &Hashed{Params: nil, Salt: make([]byte, DefaultSaltLen), Key: make([]byte, DefaultKeyLen)},
		},
		{
			name: "zero time",
			data: func() *Hashed {
				prm := validParams()
				prm.Time = 0

				return &Hashed{Params: prm, Salt: make([]byte, DefaultSaltLen), Key: make([]byte, DefaultKeyLen)}
			}(),
		},
		{
			name: "excessive time",
			data: func() *Hashed {
				prm := validParams()
				prm.Time = maxVerifyTime + 1

				return &Hashed{Params: prm, Salt: make([]byte, DefaultSaltLen), Key: make([]byte, DefaultKeyLen)}
			}(),
		},
		{
			name: "zero memory",
			data: func() *Hashed {
				prm := validParams()
				prm.Memory = 0

				return &Hashed{Params: prm, Salt: make([]byte, DefaultSaltLen), Key: make([]byte, DefaultKeyLen)}
			}(),
		},
		{
			name: "excessive memory",
			data: func() *Hashed {
				prm := validParams()
				prm.Memory = maxVerifyMemory + 1

				return &Hashed{Params: prm, Salt: make([]byte, DefaultSaltLen), Key: make([]byte, DefaultKeyLen)}
			}(),
		},
		{
			name: "zero threads",
			data: func() *Hashed {
				prm := validParams()
				prm.Threads = 0

				return &Hashed{Params: prm, Salt: make([]byte, DefaultSaltLen), Key: make([]byte, DefaultKeyLen)}
			}(),
		},
		{
			name: "zero key length",
			data: func() *Hashed {
				prm := validParams()
				prm.KeyLen = 0

				return &Hashed{Params: prm, Salt: make([]byte, DefaultSaltLen), Key: make([]byte, DefaultKeyLen)}
			}(),
		},
		{
			name: "excessive key length",
			data: func() *Hashed {
				prm := validParams()
				prm.KeyLen = maxVerifyKeyLen + 1

				return &Hashed{Params: prm, Salt: make([]byte, DefaultSaltLen), Key: make([]byte, DefaultKeyLen)}
			}(),
		},
		{
			name: "empty salt",
			data: &Hashed{Params: validParams(), Salt: nil, Key: make([]byte, DefaultKeyLen)},
		},
		{
			name: "excessive salt",
			data: &Hashed{Params: validParams(), Salt: make([]byte, maxVerifySaltLen+1), Key: make([]byte, DefaultKeyLen)},
		},
		{
			name: "declared salt length out of range",
			data: func() *Hashed {
				prm := validParams()
				prm.SaltLen = maxVerifySaltLen + 1

				return &Hashed{Params: prm, Salt: make([]byte, maxVerifySaltLen+1), Key: make([]byte, DefaultKeyLen)}
			}(),
		},
		{
			name: "key length does not match declared",
			data: &Hashed{Params: validParams(), Salt: make([]byte, DefaultSaltLen), Key: make([]byte, DefaultKeyLen-1)},
		},
		{
			name: "salt length does not match declared",
			data: &Hashed{Params: validParams(), Salt: make([]byte, DefaultSaltLen-1), Key: make([]byte, DefaultKeyLen)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			hash, err := encode.Serialize(tt.data)
			require.NoError(t, err)

			require.NotPanics(t, func() {
				ok, verr := p.PasswordVerify("test-password", hash)
				require.Error(t, verr)
				require.ErrorIs(t, verr, ErrInvalidHashData)
				require.False(t, ok)
			})
		})
	}
}

func Test_PasswordHash_PasswordVerify(t *testing.T) {
	t.Parallel()

	secret := "Test-Password-01234"

	p := New()

	hash, err := p.PasswordHash(secret)

	require.NoError(t, err)
	require.NotEmpty(t, hash)

	ok, err := p.PasswordVerify(secret, hash)

	require.NoError(t, err)
	require.True(t, ok)
}

func Test_EncryptPasswordHash(t *testing.T) {
	t.Parallel()

	p := New()

	key := []byte("0123456789012345")
	secret := "test-secret"

	hash, err := p.EncryptPasswordHash(key, secret)

	require.NoError(t, err)
	require.NotEmpty(t, hash)

	p.rnd = random.New(iotest.ErrReader(errors.New("test-rand-reader-error")))

	hash, err = p.EncryptPasswordHash(key, secret)

	require.Error(t, err)
	require.ErrorContains(t, err, "test-rand-reader-error")
	require.Empty(t, hash)
}

func Test_EncryptPasswordHash_invalidKeyLength(t *testing.T) {
	t.Parallel()

	p := New()

	secret := "test-secret"

	for _, badKey := range [][]byte{
		nil,
		[]byte(""),
		[]byte("short"),
		[]byte("012345678901234"),      // 15 bytes
		[]byte("01234567890123456789"), // 20 bytes
	} {
		hash, err := p.EncryptPasswordHash(badKey, secret)

		require.Error(t, err)
		require.Empty(t, hash)
		require.Contains(t, err.Error(), "pepper key must be 16, 24, or 32 bytes")
	}
}

func Test_EncryptPasswordVerify_invalidKeyLength(t *testing.T) {
	t.Parallel()

	p := New()

	secret := "test-secret"

	validHash, err := p.EncryptPasswordHash([]byte("0123456789012345"), secret)

	require.NoError(t, err)
	require.NotEmpty(t, validHash)

	for _, badKey := range [][]byte{
		nil,
		[]byte(""),
		[]byte("short"),
		[]byte("012345678901234"),      // 15 bytes
		[]byte("01234567890123456789"), // 20 bytes
	} {
		ok, err := p.EncryptPasswordVerify(badKey, secret, validHash)

		require.Error(t, err)
		require.False(t, ok)
		require.Contains(t, err.Error(), "pepper key must be 16, 24, or 32 bytes")
	}
}

func Test_EncryptPasswordVerify(t *testing.T) {
	t.Parallel()

	p := New()

	key := []byte("0123456789012345")
	secret := "test-secret"

	hash, err := p.EncryptPasswordHash(key, secret)

	require.NoError(t, err)
	require.NotEmpty(t, hash)

	ok, err := p.EncryptPasswordVerify(key, secret, hash)

	require.NoError(t, err)
	require.True(t, ok)

	ok, err = p.EncryptPasswordVerify(key, "wrong-secret", hash)

	require.NoError(t, err)
	require.False(t, ok)

	// Wrong pepper key: decryption fails and is classified as invalid hash data.
	ok, err = p.EncryptPasswordVerify([]byte("abcdefghijklmnop"), secret, hash)

	require.Error(t, err)
	require.ErrorIs(t, err, ErrInvalidHashData)
	require.False(t, ok)
}

func Test_validateHashParams(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*Params)
	}{
		{"nil random source", func(p *Params) { p.rnd = nil }},
		{"max password length below min", func(p *Params) { p.minPLen = 10; p.maxPLen = 5 }},
		{"algorithm not producible", func(p *Params) { p.Algo = "scrypt" }},
		{"version not producible", func(p *Params) { p.Version = 18 }},
		{"zero time", func(p *Params) { p.Time = 0 }},
		{"zero threads", func(p *Params) { p.Threads = 0 }},
		// The mint floors are stricter than the verify-time minimums: values that
		// remain verifiable for legacy hashes can no longer be minted, even by
		// mutating the exported fields directly.
		{"key length below mint floor", func(p *Params) { p.KeyLen = minHashKeyLen - 1 }},
		{"salt length below mint floor", func(p *Params) { p.SaltLen = minHashSaltLen - 1 }},
		{"memory below floor", func(p *Params) { p.Memory = 0 }},
		// The mint ceilings match the verify ceilings, so a hash can never be
		// minted with parameters its own verification would reject.
		{"time above ceiling", func(p *Params) { p.Time = maxVerifyTime + 1 }},
		{"key length above ceiling", func(p *Params) { p.KeyLen = maxVerifyKeyLen + 1 }},
		{"salt length above ceiling", func(p *Params) { p.SaltLen = maxVerifySaltLen + 1 }},
		{"memory above ceiling", func(p *Params) { p.Memory = maxVerifyMemory + 1 }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := New()
			tt.mutate(p)

			err := p.validateHashParams()

			require.Error(t, err)
			require.ErrorIs(t, err, ErrInvalidParams)
		})
	}

	// A pristine configuration validates.
	require.NoError(t, New().validateHashParams())
}

func TestPasswordHash_zeroValueParamsNoPanic(t *testing.T) {
	t.Parallel()

	// A zero-value Params (built without New) must not panic: rnd is nil, so the
	// hash path returns an error instead of dereferencing it.
	var p Params

	require.NotPanics(t, func() {
		hash, err := p.PasswordHash("some-password")

		require.ErrorIs(t, err, ErrInvalidParams)
		require.Empty(t, hash)
	})
}

func TestPasswordHash_mutatedParamsNoPanic(t *testing.T) {
	t.Parallel()

	// Mutating exported fields to values that would panic inside argon2 must be
	// caught and returned as an error instead.
	tests := []struct {
		name   string
		mutate func(*Params)
	}{
		{"zero time", func(p *Params) { p.Time = 0 }},
		{"zero threads", func(p *Params) { p.Threads = 0 }},
		{"zero key length", func(p *Params) { p.KeyLen = 0 }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := New()
			tt.mutate(p)

			require.NotPanics(t, func() {
				hash, err := p.PasswordHash("valid-password")

				require.ErrorIs(t, err, ErrInvalidParams)
				require.Empty(t, hash)
			})
		})
	}
}

func TestMintVerifyEnvelope(t *testing.T) {
	t.Parallel()

	// The invariant this whole envelope exists to guarantee: any hash the
	// package can mint, the configuration that minted it can verify. Exercise
	// it at the maximal legal parameters (key and salt at their ceilings, time
	// at its ceiling) with a small memory cost to keep the mint fast.
	p := New(
		WithKeyLen(maxVerifyKeyLen),
		WithSaltLen(maxVerifySaltLen),
		WithTime(maxVerifyTime),
		WithMemory(minMemory),
		WithThreads(1),
	)

	require.Equal(t, uint32(maxVerifyKeyLen), p.KeyLen)
	require.Equal(t, uint32(maxVerifySaltLen), p.SaltLen)
	require.Equal(t, uint32(maxVerifyTime), p.Time)

	secret := "Test-Password-01234"

	hash, err := p.PasswordHash(secret)

	require.NoError(t, err)
	// The maximal blob must still fit under the pre-decode size guard.
	require.LessOrEqual(t, len(hash), maxHashLen)

	ok, err := p.PasswordVerify(secret, hash)

	require.NoError(t, err)
	require.True(t, ok)

	// The same maximal hash verifies through the pepper-encrypted path too.
	key := []byte("0123456789012345")

	encHash, err := p.EncryptPasswordHash(key, secret)

	require.NoError(t, err)
	require.LessOrEqual(t, len(encHash), maxHashLen)

	ok, err = p.EncryptPasswordVerify(key, secret, encHash)

	require.NoError(t, err)
	require.True(t, ok)
}

func TestNew_maxPasswordLengthCrossCheck(t *testing.T) {
	t.Parallel()

	// A max below the min is raised to the min so the window never rejects
	// every possible password.
	p := New(WithMinPasswordLength(12), WithMaxPasswordLength(4))

	require.Equal(t, uint32(12), p.minPLen)
	require.Equal(t, uint32(12), p.maxPLen)

	hash, err := p.PasswordHash("abcdefghijkl") // exactly 12 bytes

	require.NoError(t, err)
	require.NotEmpty(t, hash)
}

func TestSentinelErrors(t *testing.T) {
	t.Parallel()

	p := New(WithMinPasswordLength(8), WithMaxPasswordLength(16))

	_, err := p.PasswordHash("short") // 5 < 8

	require.ErrorIs(t, err, ErrPasswordTooShort)

	_, err = p.PasswordHash(strings.Repeat("x", 17)) // 17 > 16

	require.ErrorIs(t, err, ErrPasswordTooLong)

	hash, err := p.PasswordHash("valid-password")

	require.NoError(t, err)

	wrongAlgo := New(WithMinPasswordLength(8), WithMaxPasswordLength(16))
	wrongAlgo.Algo = "wrong-algo"

	_, err = wrongAlgo.PasswordVerify("valid-password", hash)

	require.ErrorIs(t, err, ErrAlgoMismatch)

	wrongVersion := New(WithMinPasswordLength(8), WithMaxPasswordLength(16))
	wrongVersion.Version = 0

	_, err = wrongVersion.PasswordVerify("valid-password", hash)

	require.ErrorIs(t, err, ErrVersionMismatch)

	_, err = p.EncryptPasswordHash([]byte("bad-key"), "valid-password")

	require.ErrorIs(t, err, ErrInvalidPepperKey)
}

func TestPasswordNeedsRehash(t *testing.T) {
	t.Parallel()

	p := New()

	secret := "Test-Password-01234"

	hash, err := p.PasswordHash(secret)

	require.NoError(t, err)

	// Same configuration: no rehash needed.
	need, err := p.PasswordNeedsRehash(hash)

	require.NoError(t, err)
	require.False(t, need)

	// Stronger time cost: rehash needed.
	stronger := New(WithTime(p.Time + 1))

	need, err = stronger.PasswordNeedsRehash(hash)

	require.NoError(t, err)
	require.True(t, need)

	// A legacy T=1 reference hash needs a rehash under the default T=3 config.
	need, err = New().PasswordNeedsRehash(testRefHash)

	require.NoError(t, err)
	require.True(t, need)

	// Malformed hash string: decode error, classified as invalid hash data.
	need, err = p.PasswordNeedsRehash("wrong-hash")

	require.Error(t, err)
	require.ErrorIs(t, err, ErrInvalidHashData)
	require.False(t, need)

	// Well-formed encoding but invalid embedded parameters: validation error.
	badBlob, err := encode.Serialize(&Hashed{Params: &Params{}})

	require.NoError(t, err)

	need, err = p.PasswordNeedsRehash(badBlob)

	require.ErrorIs(t, err, ErrInvalidHashData)
	require.False(t, need)
}

func TestEncryptPasswordNeedsRehash(t *testing.T) {
	t.Parallel()

	p := New()

	key := []byte("0123456789012345")
	secret := "test-secret"

	hash, err := p.EncryptPasswordHash(key, secret)

	require.NoError(t, err)

	// Same configuration: no rehash needed.
	need, err := p.EncryptPasswordNeedsRehash(key, hash)

	require.NoError(t, err)
	require.False(t, need)

	// Stronger memory cost: rehash needed.
	stronger := New(WithMemory(p.Memory * 2))

	need, err = stronger.EncryptPasswordNeedsRehash(key, hash)

	require.NoError(t, err)
	require.True(t, need)

	// Invalid pepper key.
	need, err = p.EncryptPasswordNeedsRehash([]byte("bad-key"), hash)

	require.ErrorIs(t, err, ErrInvalidPepperKey)
	require.False(t, need)

	// Undecryptable payload: classified as invalid hash data.
	need, err = p.EncryptPasswordNeedsRehash(key, "not-a-valid-encrypted-hash")

	require.Error(t, err)
	require.ErrorIs(t, err, ErrInvalidHashData)
	require.False(t, need)
}

func TestOversizedHashRejected(t *testing.T) {
	t.Parallel()

	p := New()

	key := []byte("0123456789012345")
	oversized := strings.Repeat("A", maxHashLen+1)

	// All four verification/rehash entry points must reject an oversized hash
	// string before attempting any base64, JSON, or AES-GCM processing.
	ok, err := p.PasswordVerify("test-password", oversized)

	require.ErrorIs(t, err, ErrInvalidHashData)
	require.False(t, ok)

	ok, err = p.EncryptPasswordVerify(key, "test-password", oversized)

	require.ErrorIs(t, err, ErrInvalidHashData)
	require.False(t, ok)

	need, err := p.PasswordNeedsRehash(oversized)

	require.ErrorIs(t, err, ErrInvalidHashData)
	require.False(t, need)

	need, err = p.EncryptPasswordNeedsRehash(key, oversized)

	require.ErrorIs(t, err, ErrInvalidHashData)
	require.False(t, need)

	// Exactly at the limit the size guard passes and the failure happens later,
	// at the decode stage.
	ok, err = p.PasswordVerify("test-password", strings.Repeat("A", maxHashLen))

	require.ErrorIs(t, err, ErrInvalidHashData)
	require.ErrorContains(t, err, "unable to decode")
	require.False(t, ok)
}

func TestVerifyZeroValueParams(t *testing.T) {
	t.Parallel()

	// A zero-value Params on the verify and rehash-check paths must fail with a
	// clear configuration error, not the misleading "password too long: N > 0"
	// that the max-length guard would otherwise produce.
	var p Params

	ok, err := p.PasswordVerify("test-password", testRefHash)

	require.ErrorIs(t, err, ErrInvalidParams)
	require.False(t, ok)

	need, err := p.PasswordNeedsRehash(testRefHash)

	require.ErrorIs(t, err, ErrInvalidParams)
	require.False(t, need)
}
