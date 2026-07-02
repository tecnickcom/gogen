package redact

import (
	"os"
	"regexp"
	"strconv"
	"strings"
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

func TestIsSensitiveKeyASCIIFastAcronymRuns(t *testing.T) {
	t.Parallel()

	for _, key := range []string{
		"APIKey",
		"JWTToken",
		"CCNumber",
		"XPassword",
		"LASTName",
		"authorization",
		"Proxy-Authorization",
	} {
		result, ok := isSensitiveKeyASCIIFast([]byte(key))
		require.True(t, ok, key)
		require.True(t, result, key)
	}

	for _, key := range []string{
		"APIVersion",
		"IDValue",
		"Reference",
	} {
		result, ok := isSensitiveKeyASCIIFast([]byte(key))
		require.True(t, ok, key)
		require.False(t, result, key)
	}
}

func TestIsSensitiveTokenASCIIEmptyAndUnknown(t *testing.T) {
	t.Parallel()

	require.False(t, isSensitiveTokenASCII([]byte{}))
	require.False(t, isSensitiveTokenASCII([]byte("zzz")))
}

// TestSensitiveTokenSourcesInSync guards against drift between the two parallel
// representations of the sensitive-token set: the sensitiveTokens map (used by
// the normalized-key path) and the isSensitiveTokenASCII switch (the ASCII
// fast-path). Every token in the map must be recognized by the ASCII fast-path
// in every case variant, and the ASCII fast-path must not accept any extra
// single-token literal that is absent from the map.
func TestSensitiveTokenSourcesInSync(t *testing.T) {
	t.Parallel()

	for tok := range sensitiveTokens {
		// Lowercase token from the map must be accepted by the ASCII path.
		require.Truef(t, isSensitiveTokenASCII([]byte(tok)),
			"map token %q not recognized by isSensitiveTokenASCII", tok)

		// Case-insensitivity: upper- and mixed-case variants must also match.
		require.Truef(t, isSensitiveTokenASCII([]byte(strings.ToUpper(tok))),
			"upper-case variant of map token %q not recognized by isSensitiveTokenASCII", tok)

		mixed := strings.ToUpper(tok[:1]) + tok[1:]
		require.Truef(t, isSensitiveTokenASCII([]byte(mixed)),
			"mixed-case variant of map token %q not recognized by isSensitiveTokenASCII", tok)
	}

	// Reverse direction: extract every literal compared inside the ASCII
	// fast-path switch and assert the literal set equals the map key set
	// exactly. This catches a literal added to the switch but not the map.
	asciiLiterals := asciiSwitchLiterals(t)

	require.Len(t, asciiLiterals, len(sensitiveTokens),
		"sensitiveTokens map and isSensitiveTokenASCII switch have different token counts")

	for lit := range asciiLiterals {
		_, inMap := sensitiveTokens[lit]
		require.Truef(t, inMap,
			"isSensitiveTokenASCII accepts %q which is missing from the sensitiveTokens map", lit)
	}
}

// asciiSwitchLiterals parses keycache.go and returns the set of string literals
// compared via equalsASCIIFold inside the isSensitiveTokenASCII function body.
//
// The "first"/"last"/"name" comparisons live in isSensitiveKeyASCIIFast (a
// distinct multi-token special case) and are intentionally excluded.
func asciiSwitchLiterals(t *testing.T) map[string]struct{} {
	t.Helper()

	src, err := os.ReadFile("keycache.go")
	require.NoError(t, err)

	body := funcBody(t, string(src), "func isSensitiveTokenASCII(")

	re := regexp.MustCompile(`equalsASCIIFold\(tok, "([^"]+)"\)`)

	literals := make(map[string]struct{})
	for _, m := range re.FindAllStringSubmatch(body, -1) {
		literals[m[1]] = struct{}{}
	}

	require.NotEmpty(t, literals, "no equalsASCIIFold literals found in isSensitiveTokenASCII")

	return literals
}

// funcBody returns the brace-balanced body of the function whose declaration
// begins with decl, within src.
func funcBody(t *testing.T, src, decl string) string {
	t.Helper()

	start := strings.Index(src, decl)
	require.GreaterOrEqual(t, start, 0, "function %q not found", decl)

	open := strings.IndexByte(src[start:], '{')
	require.GreaterOrEqual(t, open, 0, "opening brace for %q not found", decl)

	open += start
	depth := 0

	for i := open; i < len(src); i++ {
		switch src[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return src[open : i+1]
			}
		}
	}

	require.Fail(t, "unbalanced braces", "function %q", decl)

	return ""
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
