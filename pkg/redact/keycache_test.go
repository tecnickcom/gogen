package redact

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSensitiveKeyMemoSetAndGet(t *testing.T) {
	t.Parallel()

	memo := newSensitiveKeyMemo()
	memo.set("password", true)

	v, ok := memo.get("password")
	require.True(t, ok)
	require.True(t, v)
}

func TestSensitiveKeyMemoSetEvictsBatchWhenFull(t *testing.T) {
	t.Parallel()

	memo := newSensitiveKeyMemo()

	for i := range sensitiveKeyCacheMaxEntries {
		memo.data["key"+strconv.Itoa(i)] = i%2 == 0
	}

	memo.set("newkey", true)

	// Full-cache insertion evicts a fixed batch, then inserts the new key.
	require.Len(t, memo.data, sensitiveKeyCacheMaxEntries-sensitiveKeyCacheEvictBatch+1)
	v, ok := memo.get("newkey")
	require.True(t, ok)
	require.True(t, v)
}

func TestIsSensitiveKeyASCIIFastBranches(t *testing.T) {
	t.Parallel()

	result, ok := isSensitiveKeyASCIIFast([]byte("firstNameToken"))
	require.True(t, ok)
	require.True(t, result)

	result, ok = isSensitiveKeyASCIIFast([]byte("first_name-token"))
	require.True(t, ok)
	require.True(t, result)

	result, ok = isSensitiveKeyASCIIFast([]byte("p\xc3\xa4ssword"))
	require.False(t, ok)
	require.False(t, result)
}

func TestIsSensitiveTokenASCIIEmptyAndUnknown(t *testing.T) {
	t.Parallel()

	require.False(t, isSensitiveTokenASCII([]byte{}))
	require.False(t, isSensitiveTokenASCII([]byte("zzz")))
}

func TestIsSensitiveKeyBytesFallbackPath(t *testing.T) {
	t.Parallel()

	// Non-ASCII input bypasses ASCII fast path and exercises normalize/cache fallback.
	key := []byte("p\xc3\xa4ssword")
	require.False(t, isSensitiveKeyBytes(key))
	// Second call hits memoized fallback result.
	require.False(t, isSensitiveKeyBytes(key))
}

func TestIsSensitiveNormalizedKeyDirect(t *testing.T) {
	t.Parallel()

	require.False(t, isSensitiveNormalizedKey(""))
	require.True(t, isSensitiveNormalizedKey("token"))
	require.True(t, isSensitiveNormalizedKey("first_name"))
	require.False(t, isSensitiveNormalizedKey("public_value"))
}

func TestNormalizeKeyDirect(t *testing.T) {
	t.Parallel()

	require.Equal(t, "api_key_name", normalizeKey("ApiKey Name"))
	require.Equal(t, "x_1", normalizeKey("X-1"))
}
