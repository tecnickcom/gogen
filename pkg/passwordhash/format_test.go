package passwordhash

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPasswordHash_FormatsAreCrossVerifiable(t *testing.T) {
	t.Parallel()

	secret := "Test-Password-01234"

	jsonParams := New(WithMemory(minMemory), WithTime(minTime), WithThreads(1))
	phcParams := phcTestParams()

	jsonHash, err := jsonParams.PasswordHash(secret)
	require.NoError(t, err)
	require.False(t, strings.HasPrefix(jsonHash, "$"))

	phcHash, err := phcParams.PasswordHash(secret)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(phcHash, "$"))

	// Either configuration verifies either format.
	for _, tc := range []struct {
		name string
		p    *Params
	}{
		{"json-verifier", jsonParams},
		{"phc-verifier", phcParams},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ok, verr := tc.p.PasswordVerify(secret, jsonHash)
			require.NoError(t, verr)
			require.True(t, ok)

			ok, verr = tc.p.PasswordVerify(secret, phcHash)
			require.NoError(t, verr)
			require.True(t, ok)
		})
	}
}

func TestPasswordNeedsRehash_formatConvergence(t *testing.T) {
	t.Parallel()

	secret := "Test-Password-01234"

	// Identical cost parameters in both configurations; only the serialization
	// format (and therefore the accepted-format set) differs.
	jsonParams := New(WithMemory(minMemory), WithTime(minTime), WithThreads(1))
	phcParams := phcTestParams()

	jsonHash, err := jsonParams.PasswordHash(secret)
	require.NoError(t, err)

	phcHash, err := phcParams.PasswordHash(secret)
	require.NoError(t, err)

	// Each configuration accepts its own format: no rehash for matching hashes.
	need, err := jsonParams.PasswordNeedsRehash(jsonHash)
	require.NoError(t, err)
	require.False(t, need)

	need, err = phcParams.PasswordNeedsRehash(phcHash)
	require.NoError(t, err)
	require.False(t, need)

	// A configuration accepting only its own format reports the other format as
	// needing a rehash — even with identical Argon2 parameters — so the store
	// converges to the configured format via the rehash-on-login flow.
	need, err = jsonParams.PasswordNeedsRehash(phcHash)
	require.NoError(t, err)
	require.True(t, need)

	need, err = phcParams.PasswordNeedsRehash(jsonHash)
	require.NoError(t, err)
	require.True(t, need)

	// Listing both formats as accepted keeps a deliberately mixed store: neither
	// format is flagged as long as the Argon2 parameters match.
	mixed := New(WithFormat(FormatPHC, FormatJSON), WithMemory(minMemory), WithTime(minTime), WithThreads(1))

	need, err = mixed.PasswordNeedsRehash(jsonHash)
	require.NoError(t, err)
	require.False(t, need)

	need, err = mixed.PasswordNeedsRehash(phcHash)
	require.NoError(t, err)
	require.False(t, need)
}
