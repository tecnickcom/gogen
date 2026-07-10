package passwordhash

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// testRefHashPHC is the PHC-format equivalent of testRefHash: the same 4-byte
// password "test" hashed with the same legacy parameters (T=1, M=65536, P=16),
// salt, and derived key, expressed as a standard Argon2 PHC string. It pins the
// wire format and that it decodes to the identical parameters as the JSON
// reference. (It was derived from this package's own reference, so cross-
// implementation interop is pinned separately by testExternalPHC.)
const testRefHashPHC = "$argon2id$v=19$m=65536,t=1,p=16$5wnnitUhezr1gnGhyMEU7A$BcbRTU4SCrd14bVS4sqPFbwonv+yiogOnxbV1pQLdV0"

// testExternalPHC is a frozen interop reference minted by an independent Argon2
// implementation: PHP 8's password_hash(PASSWORD_ARGON2ID) with its default cost
// parameters (m=65536, t=4, p=1). Unlike testRefHashPHC it shares no code with
// this package, so it pins true cross-implementation compatibility: a systematic
// encoding bug shared by this package's emit and parse paths would still fail
// against it.
const (
	testExternalPHCPassword = "Interop-Test-Password-01" //nolint:gosec // frozen test-vector input, not a credential
	testExternalPHC         = "$argon2id$v=19$m=65536,t=4,p=1$TlVFd2tNSVEuZjV0Ym43Qg$H5DzYw1VSOgDH4BJvB6+Gk1Vhi0qiynYGkUE0eZMC8k"
)

// phcTestParams builds a Params whose cost is cheap enough to hash quickly in
// tests while still emitting the PHC format.
func phcTestParams() *Params {
	return New(WithFormat(FormatPHC), WithMemory(minMemory), WithTime(minTime), WithThreads(1))
}

func TestPasswordHash_PHCFormat(t *testing.T) {
	t.Parallel()

	p := phcTestParams()

	secret := "Test-Password-01234"

	hash, err := p.PasswordHash(secret)

	require.NoError(t, err)
	// The emitted value is a well-formed Argon2 PHC string, not base64 JSON.
	require.True(t, strings.HasPrefix(hash, "$argon2id$v=19$"), "unexpected PHC prefix: %q", hash)
	require.Len(t, strings.Split(hash, "$"), 6)

	// It round-trips through verification regardless of the verifier's own format.
	ok, err := p.PasswordVerify(secret, hash)

	require.NoError(t, err)
	require.True(t, ok)

	ok, err = p.PasswordVerify("wrong-secret", hash)

	require.NoError(t, err)
	require.False(t, ok)

	// A default (JSON-emitting) verifier reads the PHC hash just as well: the
	// format is auto-detected from the stored value, not from the verifier.
	ok, err = New().PasswordVerify(secret, hash)

	require.NoError(t, err)
	require.True(t, ok)
}

func TestPasswordVerify_PHCReference(t *testing.T) {
	t.Parallel()

	// A default verifier accepts the externally-formatted PHC reference. The
	// minimum-length policy is not enforced on verification, so the 4-byte "test"
	// still verifies under the default min length of 8.
	p := New()

	ok, err := p.PasswordVerify("test", testRefHashPHC)

	require.NoError(t, err)
	require.True(t, ok)

	ok, err = p.PasswordVerify("wrong", testRefHashPHC)

	require.NoError(t, err)
	require.False(t, ok)
}

func TestPasswordVerify_externalPHPVector(t *testing.T) {
	t.Parallel()

	p := New()

	// The PHP-minted hash verifies with no configuration change.
	ok, err := p.PasswordVerify(testExternalPHCPassword, testExternalPHC)

	require.NoError(t, err)
	require.True(t, ok)

	ok, err = p.PasswordVerify("wrong-password-entirely", testExternalPHC)

	require.NoError(t, err)
	require.False(t, ok)

	// PHP's default costs (t=4, p=1) differ from this package's defaults, and the
	// default configuration accepts only the JSON format, so the imported hash is
	// flagged for transparent re-hashing on the next successful login.
	need, err := p.PasswordNeedsRehash(testExternalPHC)

	require.NoError(t, err)
	require.True(t, need)
}

func TestPasswordNeedsRehash_PHC(t *testing.T) {
	t.Parallel()

	p := phcTestParams()

	secret := "Test-Password-01234"

	hash, err := p.PasswordHash(secret)
	require.NoError(t, err)

	// Same configuration that minted the PHC hash: no rehash needed.
	need, err := p.PasswordNeedsRehash(hash)
	require.NoError(t, err)
	require.False(t, need)

	// A stronger configuration reports that the stored PHC hash should be upgraded.
	stronger := New(WithFormat(FormatPHC), WithMemory(minMemory), WithTime(minTime+1), WithThreads(1))

	need, err = stronger.PasswordNeedsRehash(hash)
	require.NoError(t, err)
	require.True(t, need)

	// The legacy PHC reference needs a rehash under the defaults for two
	// independent reasons: its cost parameters (T=1, P=16) are outdated, and the
	// default configuration accepts only the JSON format.
	need, err = New().PasswordNeedsRehash(testRefHashPHC)
	require.NoError(t, err)
	require.True(t, need)
}

func TestPHCMalformed(t *testing.T) {
	t.Parallel()

	p := New()

	// Every malformed PHC string (all detected by the leading '$') must be
	// rejected as invalid hash data before any Argon2 computation, never panic,
	// and never report a positive result.
	tests := []struct {
		name string
		hash string
	}{
		{"too few segments", "$argon2id$v=19$m=65536,t=1,p=16$5wnnitUhezr1gnGhyMEU7A"},
		{"too many segments", "$argon2id$v=19$m=65536,t=1,p=16$c2FsdA$aGFzaA$extra"},
		{"version prefix missing", "$argon2id$19$m=65536,t=1,p=16$c2FsdA$aGFzaA"},
		{"version not a number", "$argon2id$v=zz$m=65536,t=1,p=16$c2FsdA$aGFzaA"},
		{"version out of range", "$argon2id$v=999$m=65536,t=1,p=16$c2FsdA$aGFzaA"},
		{"cost too few fields", "$argon2id$v=19$m=65536,t=1$c2FsdA$aGFzaA"},
		{"memory prefix missing", "$argon2id$v=19$65536,t=1,p=16$c2FsdA$aGFzaA"},
		{"memory not a number", "$argon2id$v=19$m=abc,t=1,p=16$c2FsdA$aGFzaA"},
		{"time prefix missing", "$argon2id$v=19$m=65536,1,p=16$c2FsdA$aGFzaA"},
		{"time not a number", "$argon2id$v=19$m=65536,t=abc,p=16$c2FsdA$aGFzaA"},
		{"threads prefix missing", "$argon2id$v=19$m=65536,t=1,16$c2FsdA$aGFzaA"},
		{"threads out of range", "$argon2id$v=19$m=65536,t=1,p=999$c2FsdA$aGFzaA"},
		{"salt not base64", "$argon2id$v=19$m=65536,t=1,p=16$@@@@$aGFzaA"},
		{"key not base64", "$argon2id$v=19$m=65536,t=1,p=16$c2FsdA$@@@@"},
		{"memory below floor", "$argon2id$v=19$m=1,t=1,p=16$5wnnitUhezr1gnGhyMEU7A$BcbRTU4SCrd14bVS4sqPFbwonv+yiogOnxbV1pQLdV0"},
		// Canonical-form enforcement: Go's base64 decoder silently skips newlines
		// and, by default, accepts non-zero trailing padding bits, so without the
		// explicit guards these byte-distinct strings would verify successfully.
		{"trailing newline", testRefHashPHC + "\n"},
		{"embedded newline in key", testRefHashPHC[:len(testRefHashPHC)-4] + "\n" + testRefHashPHC[len(testRefHashPHC)-4:]},
		{"carriage return", testRefHashPHC + "\r"},
		{"non-canonical base64 trailing bits", testRefHashPHC[:len(testRefHashPHC)-1] + "1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.NotPanics(t, func() {
				ok, err := p.PasswordVerify("test-password", tt.hash)
				require.Error(t, err)
				require.ErrorIs(t, err, ErrInvalidHashData)
				require.False(t, ok)

				need, rerr := p.PasswordNeedsRehash(tt.hash)
				require.Error(t, rerr)
				require.ErrorIs(t, rerr, ErrInvalidHashData)
				require.False(t, need)
			})
		})
	}
}

func TestPHCAlgoAndVersionMismatch(t *testing.T) {
	t.Parallel()

	p := New()

	// A structurally valid PHC string carrying a different algorithm or version
	// is classified as a mismatch (a migration signal), not as corrupt data.
	ok, err := p.PasswordVerify("test", "$argon2i$v=19$m=65536,t=1,p=16$5wnnitUhezr1gnGhyMEU7A$BcbRTU4SCrd14bVS4sqPFbwonv+yiogOnxbV1pQLdV0")

	require.ErrorIs(t, err, ErrAlgoMismatch)
	require.False(t, ok)

	ok, err = p.PasswordVerify("test", "$argon2id$v=16$m=65536,t=1,p=16$5wnnitUhezr1gnGhyMEU7A$BcbRTU4SCrd14bVS4sqPFbwonv+yiogOnxbV1pQLdV0")

	require.ErrorIs(t, err, ErrVersionMismatch)
	require.False(t, ok)
}

func TestEncryptPathRejectsPHC(t *testing.T) {
	t.Parallel()

	// The pepper-encrypted methods do not accept PHC (or any plaintext) input:
	// their stored value is always an AES-GCM ciphertext, so a PHC string fails
	// authenticated decryption and is reported as invalid hash data.
	p := New()

	key := []byte("0123456789012345")

	ok, err := p.EncryptPasswordVerify(key, "test", testRefHashPHC)

	require.ErrorIs(t, err, ErrInvalidHashData)
	require.False(t, ok)

	need, err := p.EncryptPasswordNeedsRehash(key, testRefHashPHC)

	require.ErrorIs(t, err, ErrInvalidHashData)
	require.False(t, need)
}
