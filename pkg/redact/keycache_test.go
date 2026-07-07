package redact

import (
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

	result, ok := defaultRedactor.sensitiveKeyASCIIFast([]byte("firstNameToken"))
	require.True(t, ok)
	require.True(t, result)

	result, ok = defaultRedactor.sensitiveKeyASCIIFast([]byte("first_name-token"))
	require.True(t, ok)
	require.True(t, result)

	result, ok = defaultRedactor.sensitiveKeyASCIIFast([]byte("p\xc3\xa4ssword"))
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
		result, ok := defaultRedactor.sensitiveKeyASCIIFast([]byte(key))
		require.True(t, ok, key)
		require.True(t, result, key)
	}

	for _, key := range []string{
		"APIVersion",
		"IDValue",
		"Reference",
	} {
		result, ok := defaultRedactor.sensitiveKeyASCIIFast([]byte(key))
		require.True(t, ok, key)
		require.False(t, result, key)
	}
}

func TestIsSensitiveTokenASCIIEmptyAndUnknown(t *testing.T) {
	t.Parallel()

	require.False(t, defaultRedactor.sensitiveTokenASCII([]byte{}))
	require.False(t, defaultRedactor.sensitiveTokenASCII([]byte("zzz")))
}

// TestSensitiveTokensLookup verifies that every token in the sensitiveTokens
// map (the single source of truth) is matched case-insensitively by the
// allocation-free ASCII lookup, and that the length bound assumed by the
// lookup's stack buffer holds for all tokens.
func TestSensitiveTokensLookup(t *testing.T) {
	t.Parallel()

	for tok := range sensitiveTokens {
		// Every token must fit the lookup's fixed lowercase buffer.
		require.LessOrEqualf(t, len(tok), maxSensitiveTokenLen,
			"token %q exceeds maxSensitiveTokenLen; increase the bound", tok)

		// Lowercase token from the map must be accepted by the ASCII path.
		require.Truef(t, defaultRedactor.sensitiveTokenASCII([]byte(tok)),
			"map token %q not recognized by isSensitiveTokenASCII", tok)

		// Case-insensitivity: upper- and mixed-case variants must also match.
		require.Truef(t, defaultRedactor.sensitiveTokenASCII([]byte(strings.ToUpper(tok))),
			"upper-case variant of map token %q not recognized by isSensitiveTokenASCII", tok)

		mixed := strings.ToUpper(tok[:1]) + tok[1:]
		require.Truef(t, defaultRedactor.sensitiveTokenASCII([]byte(mixed)),
			"mixed-case variant of map token %q not recognized by isSensitiveTokenASCII", tok)
	}

	// Candidates longer than any token are rejected before the map lookup.
	require.False(t, defaultRedactor.sensitiveTokenASCII([]byte(strings.Repeat("a", maxSensitiveTokenLen+1))))
}

func TestIsSensitiveKeyBytesFallbackPath(t *testing.T) {
	t.Parallel()

	// Non-ASCII input bypasses ASCII fast path and exercises normalize/cache fallback.
	key := []byte("p\xc3\xa4ssword")
	require.False(t, defaultRedactor.isSensitiveKey(key))
	// Second call hits memoized fallback result.
	require.False(t, defaultRedactor.isSensitiveKey(key))
}

func TestIsSensitiveNormalizedKeyDirect(t *testing.T) {
	t.Parallel()

	require.False(t, defaultRedactor.isSensitiveNormalizedKeyTokens(""))
	require.True(t, defaultRedactor.isSensitiveNormalizedKeyTokens("token"))
	require.True(t, defaultRedactor.isSensitiveNormalizedKeyTokens("first_name"))
	require.False(t, defaultRedactor.isSensitiveNormalizedKeyTokens("public_value"))
}

func TestNormalizeKeyDirect(t *testing.T) {
	t.Parallel()

	require.Equal(t, "api_key_name", normalizeKey("ApiKey Name"))
	require.Equal(t, "x_1", normalizeKey("X-1"))
}

func TestHTTPDataKeywordBoundaries(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  string
	}{
		// no false-positive substring match for short keyword fragments
		{`{"access_log": "VISIBLE", "monkey": "VISIBLE"}`, `{"access_log": "VISIBLE", "monkey": "VISIBLE"}`},
		// old broad "user" matching is removed
		{`{"user_agent": "VISIBLE"}`, `{"user_agent": "VISIBLE"}`},
		// token-based sensitive keys still redact
		{`{"apiKey": "SECRET", "acc_number": "SECRET", "firstName": "SECRET"}`, `{"apiKey": "***", "acc_number": "***", "firstName": "***"}`},
		{`access_log=VISIBLE&monkey=VISIBLE`, `access_log=VISIBLE&monkey=VISIBLE`},
		{`apiKey=SECRET&acc_number=SECRET&firstName=SECRET`, `apiKey=***&acc_number=***&firstName=***`},
	}

	for _, tc := range cases {
		require.Equal(t, expectedRedaction(tc.want), HTTPData(tc.input), "input: %s", tc.input)
	}
}

func TestIsSensitiveNormalizedKeyEmpty(t *testing.T) {
	t.Parallel()

	require.False(t, defaultRedactor.isSensitiveNormalizedKeyTokens(""))
}

func TestHTTPDataAcronymRunKeys(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  string
	}{
		{`{"APIKey":"SECRET"}`, `{"APIKey":"***"}`},
		{`{"JWTToken":"SECRET"}`, `{"JWTToken":"***"}`},
		{`{"CCNumber":"SECRET"}`, `{"CCNumber":"***"}`},
		{`{"XPassword":"SECRET"}`, `{"XPassword":"***"}`},
		{`APIKey=SECRET&reference=VISIBLE`, `APIKey=***&reference=VISIBLE`},
	}

	for _, tc := range cases {
		require.Equal(t, expectedRedaction(tc.want), HTTPData(tc.input), "input: %s", tc.input)
	}
}

func TestNormalizeKeyAcronymRuns(t *testing.T) {
	t.Parallel()

	require.Equal(t, "api_key", normalizeKey("APIKey"))
	require.Equal(t, "jwt_token", normalizeKey("JWTToken"))
	require.Equal(t, "cc_number", normalizeKey("CCNumber"))
	require.Equal(t, "dsn", normalizeKey("DSN"))
	require.Equal(t, "x_password", normalizeKey("XPassword"))
}

// TestHTTPDataClosedCompoundKeys covers closed-compound (concatenated, lowercase)
// key names that previously leaked because token matching is exact-word.
func TestHTTPDataClosedCompoundKeys(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  string
	}{
		{`{"apikey":"SECRET"}`, `{"apikey":"***"}`},
		{`apikey=SECRET`, `apikey=***`},
		{`{"credential":"SECRET"}`, `{"credential":"***"}`},
		{`{"credentials":"SECRET"}`, `{"credentials":"***"}`},
		{`{"passcode":"SECRET"}`, `{"passcode":"***"}`},
		{`{"passphrase":"SECRET"}`, `{"passphrase":"***"}`},
		{`{"secretkey":"SECRET"}`, `{"secretkey":"***"}`},
		{`{"privatekey":"SECRET"}`, `{"privatekey":"***"}`},
		{`{"accesstoken":"SECRET"}`, `{"accesstoken":"***"}`},
		{`{"authtoken":"SECRET"}`, `{"authtoken":"***"}`},
		{`{"pin":"SECRET"}`, `{"pin":"***"}`},
		{`{"otp":"SECRET"}`, `{"otp":"***"}`},
		{`{"totp":"SECRET"}`, `{"totp":"***"}`},
		{`{"mfa":"SECRET"}`, `{"mfa":"***"}`},
		{`{"signature":"SECRET"}`, `{"signature":"***"}`},
		{`{"clientsecret":"SECRET"}`, `{"clientsecret":"***"}`},
		{`{"refreshtoken":"SECRET"}`, `{"refreshtoken":"***"}`},
		{`{"idtoken":"SECRET"}`, `{"idtoken":"***"}`},
		{`{"accesskey":"SECRET"}`, `{"accesskey":"***"}`},
		{`{"apisecret":"SECRET"}`, `{"apisecret":"***"}`},
		{`{"appsecret":"SECRET"}`, `{"appsecret":"***"}`},
		{`{"bearertoken":"SECRET"}`, `{"bearertoken":"***"}`},
		{`{"sessionid":"SECRET"}`, `{"sessionid":"***"}`},
		{`{"sessionkey":"SECRET"}`, `{"sessionkey":"***"}`},
		{`{"sessiontoken":"SECRET"}`, `{"sessiontoken":"***"}`},
		{`{"csrf":"SECRET"}`, `{"csrf":"***"}`},
		{`{"csrftoken":"SECRET"}`, `{"csrftoken":"***"}`},
		{`{"xsrf":"SECRET"}`, `{"xsrf":"***"}`},
		{`{"xsrftoken":"SECRET"}`, `{"xsrftoken":"***"}`},
		{`{"nonce":"SECRET"}`, `{"nonce":"***"}`},
		{`PHPSESSID=SECRET`, `PHPSESSID=***`},
		{`JSESSIONID=SECRET`, `JSESSIONID=***`},
		{`X-Amz-Signature=SECRET`, `X-Amz-Signature=***`},
		// Near-miss words must remain visible: exact-token matching, not substring.
		{`{"monkey":"VISIBLE"}`, `{"monkey":"VISIBLE"}`},
		{`{"spinner":"VISIBLE"}`, `{"spinner":"VISIBLE"}`},
		{`{"pinboard":"VISIBLE"}`, `{"pinboard":"VISIBLE"}`},
		{`{"apikeys":"VISIBLE"}`, `{"apikeys":"VISIBLE"}`},
		{`{"signatures":"VISIBLE"}`, `{"signatures":"VISIBLE"}`},
		{`{"announced":"VISIBLE"}`, `{"announced":"VISIBLE"}`},
	}

	for _, tc := range cases {
		require.Equal(t, expectedRedaction(tc.want), HTTPData(tc.input), "input: %s", tc.input)
	}
}

// FuzzKeyClassificationParity asserts the ASCII fast path and the
// normalize+lookup fallback classify every ASCII key identically, guarding
// against future drift between the two implementations.
func FuzzKeyClassificationParity(f *testing.F) {
	seeds := []string{
		"password", "apiKey", "APIKey", "JWTToken", "CCNumber", "IDToken",
		"keyID", "firstName", "LASTName", "first_name", "last-name",
		"X-1", "access_log", "monkey", "user_agent", "apikey", "credential",
		"AccessToken", "x_api_key", "", "a", "A", "1", "_", "aA", "Aa",
		"ABc", "aBC", "camelCaseKey", "snake_case_key", "SCREAMING_SNAKE",
		"first-name-token", "lastName", "pin", "otp", "mfa", "secretKey",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, key string) {
		fast, ok := defaultRedactor.sensitiveKeyASCIIFast([]byte(key))
		if !ok {
			return // non-ASCII: the fast path defers to the fallback; nothing to compare.
		}

		slow := defaultRedactor.isSensitiveNormalizedKeyTokens(normalizeKey(key))
		require.Equalf(t, slow, fast, "divergence for key %q (normalized %q)", key, normalizeKey(key))
	})
}

// TestEnvVarCompoundTokens covers ALLCAPS closed-compound environment secrets.
func TestEnvVarCompoundTokens(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  string
	}{
		{"PGPASSWORD=secret", "PGPASSWORD=***"},
		{"export PGPASSWORD=secret", "export PGPASSWORD=***"},
		{"HTPASSWD=x", "HTPASSWD=***"},
		{`{"dbpassword":"S"}`, `{"dbpassword":"***"}`},
		{`{"connectionstring":"S"}`, `{"connectionstring":"***"}`},
	}

	for _, tc := range cases {
		require.Equal(t, expectedRedaction(tc.want), HTTPData(tc.input), "input: %s", tc.input)
	}
}
