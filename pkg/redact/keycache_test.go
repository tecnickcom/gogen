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
	memo.data = make(map[string]bool, sensitiveKeyCacheMaxEntries) // the backing map is lazily allocated

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
		// Plurals of a sensitive token are the same secret.
		{`{"apikeys":"SECRET"}`, `{"apikeys":"***"}`},
		{`{"accesskeys":"SECRET"}`, `{"accesskeys":"***"}`},
		{`{"signatures":"SECRET"}`, `{"signatures":"***"}`},
		{`{"tokens":"SECRET"}`, `{"tokens":"***"}`},
		{`{"passwords":"SECRET"}`, `{"passwords":"***"}`},
		// Near-miss words must remain visible: token matching, not substring.
		{`{"monkey":"VISIBLE"}`, `{"monkey":"VISIBLE"}`},
		{`{"spinner":"VISIBLE"}`, `{"spinner":"VISIBLE"}`},
		{`{"pinboard":"VISIBLE"}`, `{"pinboard":"VISIBLE"}`},
		{`{"announced":"VISIBLE"}`, `{"announced":"VISIBLE"}`},
		{`{"wildcard":"VISIBLE"}`, `{"wildcard":"VISIBLE"}`},
		{`{"discard":"VISIBLE"}`, `{"discard":"VISIBLE"}`},
		{`{"tokenizer":"VISIBLE"}`, `{"tokenizer":"VISIBLE"}`},
		{`{"passwordless":"VISIBLE"}`, `{"passwordless":"VISIBLE"}`},
		{`{"status":"VISIBLE"}`, `{"status":"VISIBLE"}`},
	}

	for _, tc := range cases {
		require.Equal(t, expectedRedaction(tc.want), HTTPData(tc.input), "input: %s", tc.input)
	}
}

// TestGluedCompoundKeys covers all-lowercase closed compounds that have no
// case or separator boundary to tokenize on ("newpassword", the shape HTML
// forms use). They are matched by the bounded root-suffix rule, not by
// substring search: the near-miss cases in TestHTTPDataClosedCompoundKeys pin
// the other side of that boundary.
func TestGluedCompoundKeys(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  string
	}{
		// The libpq / MySQL / LDAP connection parameter and web form field.
		{`passwd=hunter2`, `passwd=***`},
		{`{"passwd":"hunter2"}`, `{"passwd":"***"}`},
		{"Passwd: hunter2\r\n", "Passwd: ***\r\n"},
		// URL-encoded form dumps of <input name="newpassword">.
		{`oldpassword=hunter2&newpassword=s3cr3t`, `oldpassword=***&newpassword=***`},
		{`{"newpassword":"hunter2"}`, `{"newpassword":"***"}`},
		{`{"userpassword":"S"}`, `{"userpassword":"***"}`},
		{`{"rootpassword":"S"}`, `{"rootpassword":"***"}`},
		{`{"adminpassword":"S"}`, `{"adminpassword":"***"}`},
		{`{"smtppassword":"S"}`, `{"smtppassword":"***"}`},
		{`{"dbpasswd":"S"}`, `{"dbpasswd":"***"}`},
		{`{"newpwd":"S"}`, `{"newpwd":"***"}`},
		// Card names.
		{`{"creditcard":"S"}`, `{"creditcard":"***"}`},
		{`{"debitcard":"S"}`, `{"debitcard":"***"}`},
		{`{"cardnumber":"S"}`, `{"cardnumber":"***"}`},
		{`{"creditcardnumber":"S"}`, `{"creditcardnumber":"***"}`},
		// The *key family: "key" alone is too collision-prone to be a root, so
		// each unambiguous compound is.
		{`{"authkey":"S"}`, `{"authkey":"***"}`},
		{`{"privkey":"S"}`, `{"privkey":"***"}`},
		{`{"sshkey":"S"}`, `{"sshkey":"***"}`},
		{`{"signingkey":"S"}`, `{"signingkey":"***"}`},
		{`{"encryptionkey":"S"}`, `{"encryptionkey":"***"}`},
		{`{"awssecretkey":"S"}`, `{"awssecretkey":"***"}`},
		{`{"rsaprivatekey":"S"}`, `{"rsaprivatekey":"***"}`},
		{`{"openaiapikey":"S"}`, `{"openaiapikey":"***"}`},
		// Other roots.
		{`{"webhooksecret":"S"}`, `{"webhooksecret":"***"}`},
		{`{"apitoken":"S"}`, `{"apitoken":"***"}`},
		{`{"hmacsignature":"S"}`, `{"hmacsignature":"***"}`},
		{`{"awscredential":"S"}`, `{"awscredential":"***"}`},
		// A glued token longer than the stack buffer still reaches the
		// length-agnostic root-suffix rule.
		{`{"averyveryverylongadministratorpassword":"S"}`, `{"averyveryverylongadministratorpassword":"***"}`},
	}

	for _, tc := range cases {
		require.Equal(t, expectedRedaction(tc.want), HTTPData(tc.input), "input: %s", tc.input)
	}
}

// TestSensitiveRootsAreTokens locks the invariants the root-suffix rule assumes:
// every root is also an exact token, so hasSensitiveRootSuffix can skip the
// equal-length case and leave it to the exact lookup (which honors
// WithoutTokens); and every root really does match as a suffix, so a root added
// below minGluedCompoundLen cannot be silently skipped by the length gate.
func TestSensitiveRootsAreTokens(t *testing.T) {
	t.Parallel()

	for _, root := range sensitiveRoots {
		require.Containsf(t, sensitiveTokens, root, "root %q is not an exact sensitive token", root)
		require.LessOrEqualf(t, len(root), maxSensitiveTokenLen, "root %q exceeds maxSensitiveTokenLen", root)
		require.Truef(t, defaultRedactor.sensitiveTokenASCII([]byte(root)), "root %q not recognized", root)

		// A compound built on the root is matched as a whole token, and its
		// plural too. The "test" prefix keeps every candidate at or above
		// minGluedCompoundLen.
		compound := "test" + root
		require.Truef(t, defaultRedactor.sensitiveTokenASCII([]byte(compound)),
			"compound %q not matched by the root-suffix rule", compound)
		require.Truef(t, defaultRedactor.sensitiveTokenASCII([]byte(compound+"s")),
			"plural compound %qs not matched", compound)
	}
}

// TestKeyMatchingRetriesAndCompounds locks the key-classification behavior: the
// digit- and plural-strip retries, the short-root length gate, the *key/*cookie
// and payment/PII tokens, and long glued tokens — together with the "stays
// visible" side each of these must not regress.
func TestKeyMatchingRetriesAndCompounds(t *testing.T) {
	t.Parallel()

	redact := []string{
		// a trailing digit run marks a numbered field.
		"password2", "password1", "cvv2", "apikey2", "key1", "pin2", "secret2",
		"token2", "passwd2", "otp1", "pwd1",
		// short pwd compounds that do not tokenize on their own.
		"dbpwd", "mypwd", "newpwd", "oldpwd",
		// *key / *cookie glued compounds.
		"masterkey", "deploykey", "hostkey", "gpgkey", "signkey", "rootkey",
		"sessioncookie", "authcookie",
		// payment / PII tokens.
		"cvn", "csc", "passport", "passportnumber", "nationalid",
		// plurals of the strong roots.
		"tokens", "apikeys", "accesskeys", "passwords", "secrets", "signatures",
		"newpasswords",
		// a glued token past the stack buffer.
		"administratoraccountrecoverypassword",
	}
	for _, k := range redact {
		require.Truef(t, defaultRedactor.isSensitiveKey([]byte(k)), "expected %q to be sensitive", k)
	}

	visible := []string{
		// plurals of ordinary-word exact tokens must NOT be swept up —
		// "keys" is the JWKS case, the rest are common non-secret arrays.
		"keys", "cells", "accounts", "payments", "emails", "phones", "bills",
		"cards", "amounts", "balances", "logins", "sessions", "pins",
		// digit-suffixed words whose base is not a token must stay visible
		// (hashes, encodings, curve names, resource ordinals).
		"sha256", "base64", "utf8", "x509", "oauth2", "region2", "node3",
		"shard2", "md5", "argon2", "p256", "route53", "ec2", "http2", "s3",
		// Pre-existing near-miss guarantees must not regress.
		"monkey", "wildcard", "discard", "tokenizer", "passwordless", "status",
		"donkey", "turkey", "whiskey", "apikeyx",
	}
	for _, k := range visible {
		require.Falsef(t, defaultRedactor.isSensitiveKey([]byte(k)), "expected %q to stay visible", k)
	}

	// The JSON rule replaces whole containers, so a false-positive plural key
	// would destroy an entire array; a JWKS response must survive intact.
	jwks := `{"keys":[{"kty":"RSA","n":"abc"}]}`
	require.Equal(t, jwks, String(jwks))
}

// TestKeyMatchingConfigInteractions locks the coherence of the three matching
// modes with WithExtraTokens / WithoutTokens.
func TestKeyMatchingConfigInteractions(t *testing.T) {
	t.Parallel()

	// Dropping a root disables its exact, digit, plural, and suffix forms together.
	drop := New(WithoutTokens("password"))
	for _, k := range []string{"password", "password2", "passwords", "newpassword"} {
		require.Falsef(t, drop.isSensitiveKey([]byte(k)), "WithoutTokens(password) should hide %q", k)
	}
	// A different root still redacts on the same instance.
	require.True(t, drop.isSensitiveKey([]byte("apikey")))

	// An extra token gains the digit retry but NOT the roots-only plural retry
	// (extra tokens are not roots).
	extra := New(WithExtraTokens("floof"))
	require.True(t, extra.isSensitiveKey([]byte("floof2")))   // digit retry over the exact set
	require.False(t, extra.isSensitiveKey([]byte("myfloof"))) // extra tokens are not suffix roots
	require.False(t, extra.isSensitiveKey([]byte("floofs")))  // plural retry is roots-only
	require.False(t, defaultRedactor.isSensitiveKey([]byte("floof")))
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
		// digit/plural/long-token retries must agree on both paths too.
		"password2", "cvv2", "key1", "keys", "tokens", "apikeys", "cells",
		"dbpwd", "MyPwd", "sha256", "MasterKey", "PASSWORD2", "newpasswords",
		"administratoraccountrecoverypassword", "authcookie", "passport",
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

// TestSensitiveTokenSuffixEdges covers the digit- and plural-strip retries that
// reduce to an empty or non-matching base.
func TestSensitiveTokenSuffixEdges(t *testing.T) {
	t.Parallel()

	// An all-digit token strips to an empty base and must not match.
	require.False(t, defaultRedactor.sensitiveToken("123"))
	// A plural whose singular is neither a root nor a compound stays visible.
	require.False(t, defaultRedactor.sensitiveToken("cells"))
	// A lone "s" must not match (empty base after the plural strip).
	require.False(t, defaultRedactor.sensitiveToken("s"))
}

// TestEmptyTokenGuards pins the empty-token guards of the two token entry
// points. The tokenizer never emits an empty token, so these are defensive:
// they keep a future caller (or a hand-built token) from indexing an empty
// string or matching the root set on nothing.
func TestEmptyTokenGuards(t *testing.T) {
	t.Parallel()

	require.False(t, defaultRedactor.sensitiveToken(""))
	require.False(t, defaultRedactor.matchesRoot(""))
	require.False(t, defaultRedactor.matchesTokenOrRoot(""))
}

// TestWithoutTokensPluralCoherence covers the drop-token branch of the
// roots-only plural retry.
func TestWithoutTokensPluralCoherence(t *testing.T) {
	t.Parallel()

	re := New(WithoutTokens("token"))
	require.False(t, re.isSensitiveKey([]byte("token")))
	require.False(t, re.isSensitiveKey([]byte("tokens")))   // plural retry honors the drop
	require.False(t, re.isSensitiveKey([]byte("mytoken")))  // suffix rule honors the drop
	require.True(t, re.isSensitiveKey([]byte("authtoken"))) // a separate exact token is unaffected
	require.True(t, re.isSensitiveKey([]byte("password")))  // other roots unaffected
}
