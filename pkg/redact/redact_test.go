package redact

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func expectedRedaction(s string) string {
	return strings.ReplaceAll(s, "***", RedactionMarker)
}

func TestRedactionMarkerValue(t *testing.T) {
	t.Parallel()

	require.Equal(t, "***", RedactionMarker)
}

const benchmarkHTTPDataInput = `
POST /v1/login HTTP/1.1
Host: test.redact.invalid
Authorization: Bearer SECRET_TOKEN
Content-Type: application/json

{"password":"SECRET","apiKey":"SECRET","reference":"VISIBLE","card":"4012888888881881"}
`

func TestHTTPData(t *testing.T) {
	t.Parallel()

	data := `
GET /v1/version HTTP/1.1
Host: test.redact.invalid
User-Agent: Go-http-client/1.1
Authorization: Basic SECRET_ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789=
authorization : ApiKey=SECRET OtherData=SECRET
AUTHORIZATION	:	Digest SECRET
X-GOGEN-Trace-Id: abcdef0123456789
Accept-Encoding: gzip

password=SECRET
test_password=SECRET
PASSWORD=SECRET
TEST_PASSWORD=SECRET
key=SECRET
test_key=SECRET
KEY=SECRET
TEST_KEY=SECRET
password=SECRET&key=SECRET
ApiKey=SECRET&alpha=beta&password=SECRET&key=SECRET&gamma=delta
Token=SECRET
security_key=SECRET
secure_data=SECRET
auth_token=SECRET
bearer_token=SECRET
cert_chain=SECRET
cookie_data=SECRET
cred_value=SECRET
dsn=SECRET
jwt_token=SECRET
login_user=SECRET
pwd_hint=SECRET
seal_code=SECRET
secret_value=SECRET
session_id=SECRET
sgn_payload=SECRET
sid_value=SECRET
sig_value=SECRET
checksum=SECRET
dsa_signature=SECRET
ecdsa_signature=SECRET
fingerprint=SECRET
hash_value=SECRET
hmac_value=SECRET
pkcs12=SECRET
proof_data=SECRET
rsa_key=SECRET
salt_value=SECRET
attestation=SECRET
autograph=SECRET
endorse_token=SECRET
acc_number=SECRET
amount_due=SECRET
bal_total=SECRET
bill_ref=SECRET
card_holder=SECRET
cc_number=SECRET
cv2_code=SECRET
cvc_code=SECRET
cvv_code=SECRET
iban_code=SECRET
pay_ref=SECRET
swift_code=SECRET
addr_line=SECRET
birth_date=SECRET
cell_number=SECRET
dob_value=SECRET
email_addr=SECRET
expiry_date=SECRET
holder_name=SECRET
mail_box=SECRET
phone_number=SECRET
postal_code=SECRET
social_security=SECRET
ssn_code=SECRET
tax_id=SECRET
tel_number=SECRET
first_name=SECRET
last-name=SECRET
reference=VISIBLE
note=VISIBLE
Visa13=(4222222222222)
Visa16=(4012888888881881)
MasterCard2Series=(2223000048400011)
MasterCard5Series=(5555555555554444)
Discover6011=(6011000990139424)
Discover65Series=(6500000000000002)
Amex34=(341111111111111)
Amex37=(371449635398431)
Diners30=(30569309025904)
Diners38=(38520000023237)
JCB2131=(213100000000008)
JCB1800=(180000000000002)
JCB35=(3566002020360505)

{
	"password":"SECRET",
	"Password": "SECRET",
	"password" : "SECRET","password" :"SECRET",
	"test_password":"SECRET",
	"test_password_test": "SECRET",
	"test_password" : "SECRET","test_password" :"SECRET",
	"PASSWORD":"SECRET",
	"PASSWORD": "SECRET",
	"PASSWORD" : "SECRET","PASSWORD" :"SECRET",
	"TEST_PASSWORD":"SECRET",
	"TEST_PASSWORD": "SECRET",
	"TEST_PASSWORD" : "SECRET","TEST_PASSWORD" :"SECRET",
	"key":"SECRET",
	"Key": "SECRET",
	"key" : "SECRET","key" :"SECRET",
	"test_key":"SECRET",
	"test_key": "SECRET",
	"test_key" : "SECRET","test_key" :"SECRET",
	"KEY":"SECRET",
	"KEY": "SECRET",
	"KEY" : "SECRET","KEY" :"SECRET",
	"TEST_KEY":"SECRET",
	"TEST_KEY": "SECRET",
	"TEST_KEY" : "SECRET","TEST_KEY" :"SECRET",
	"ApiKey":"SECRET",
	"ApiKey": "SECRET",
	"ApiKey" : "SECRET","ApiKey" :"SECRET",
	"Token" : "SECRET",
	"OtherField" : "OtherValue",
    "Visa" : "4012888888881881",
    "MasterCard" : "5555555555554444",
    "American Express" : "371449635398431",
    "Diners Club" : "38520000023237",
    "Discover" : "6011000990139424",
    "JCB" : "3566002020360505",
	"Security_Key": "SECRET",
	"Secure_Data": "SECRET",
	"Auth_Token": "SECRET",
	"BearerToken": "SECRET",
	"CertChain": "SECRET",
	"CookieData": "SECRET",
	"CredValue": "SECRET",
	"DSN": "SECRET",
	"JWT_Token": "SECRET",
	"LoginUser": "SECRET",
	"PwdHint": "SECRET",
	"SealCode": "SECRET",
	"SecretValue": "SECRET",
	"SessionId": "SECRET",
	"SgnPayload": "SECRET",
	"SidValue": "SECRET",
	"SigValue": "SECRET",
	"Checksum": "SECRET",
	"DSA_Signature": "SECRET",
	"ECDSA_Signature": "SECRET",
	"Fingerprint": "SECRET",
	"HashValue": "SECRET",
	"HmacValue": "SECRET",
	"PKCS12": "SECRET",
	"ProofData": "SECRET",
	"RSA_Key": "SECRET",
	"SaltValue": "SECRET",
	"Attestation": "SECRET",
	"Autograph": "SECRET",
	"EndorseToken": "SECRET",
	"AccNumber": "SECRET",
	"AmountDue": "SECRET",
	"BalTotal": "SECRET",
	"BillRef": "SECRET",
	"CardHolder": "SECRET",
	"CC_Number": "SECRET",
	"CV2_Code": "SECRET",
	"CVC_Code": "SECRET",
	"CVV_Code": "SECRET",
	"IBAN_Code": "SECRET",
	"PayRef": "SECRET",
	"SwiftCode": "SECRET",
	"AddrLine": "SECRET",
	"BirthDate": "SECRET",
	"CellNumber": "SECRET",
	"DobValue": "SECRET",
	"EmailAddr": "SECRET",
	"ExpiryDate": "SECRET",
	"HolderName": "SECRET",
	"MailBox": "SECRET",
	"PhoneNumber": "SECRET",
	"PostalCode": "SECRET",
	"SocialSecurity": "SECRET",
	"SSN_Code": "SECRET",
	"TaxId": "SECRET",
	"TelNumber": "SECRET",
	"First Name": "SECRET",
	"Last-Name": "SECRET",
	"Reference": "VISIBLE"
}
`
	expected := `
GET /v1/version HTTP/1.1
Host: test.redact.invalid
User-Agent: Go-http-client/1.1
Authorization: ***
authorization : ***
AUTHORIZATION	:	***
X-GOGEN-Trace-Id: abcdef0123456789
Accept-Encoding: gzip

password=***
test_password=***
PASSWORD=***
TEST_PASSWORD=***
key=***
test_key=***
KEY=***
TEST_KEY=***
password=***&key=***
ApiKey=***&alpha=beta&password=***&key=***&gamma=delta
Token=***
security_key=***
secure_data=***
auth_token=***
bearer_token=***
cert_chain=***
cookie_data=***
cred_value=***
dsn=***
jwt_token=***
login_user=***
pwd_hint=***
seal_code=***
secret_value=***
session_id=***
sgn_payload=***
sid_value=***
sig_value=***
checksum=***
dsa_signature=***
ecdsa_signature=***
fingerprint=***
hash_value=***
hmac_value=***
pkcs12=***
proof_data=***
rsa_key=***
salt_value=***
attestation=***
autograph=***
endorse_token=***
acc_number=***
amount_due=***
bal_total=***
bill_ref=***
card_holder=***
cc_number=***
cv2_code=***
cvc_code=***
cvv_code=***
iban_code=***
pay_ref=***
swift_code=***
addr_line=***
birth_date=***
cell_number=***
dob_value=***
email_addr=***
expiry_date=***
holder_name=***
mail_box=***
phone_number=***
postal_code=***
social_security=***
ssn_code=***
tax_id=***
tel_number=***
first_name=***
last-name=***
reference=VISIBLE
note=VISIBLE
Visa13=(***)
Visa16=(***)
MasterCard2Series=(***)
MasterCard5Series=(***)
Discover6011=(***)
Discover65Series=(***)
Amex34=(***)
Amex37=(***)
Diners30=(***)
Diners38=(***)
JCB2131=(***)
JCB1800=(***)
JCB35=(***)

{
	"password":"***",
	"Password": "***",
	"password" : "***","password" :"***",
	"test_password":"***",
	"test_password_test": "***",
	"test_password" : "***","test_password" :"***",
	"PASSWORD":"***",
	"PASSWORD": "***",
	"PASSWORD" : "***","PASSWORD" :"***",
	"TEST_PASSWORD":"***",
	"TEST_PASSWORD": "***",
	"TEST_PASSWORD" : "***","TEST_PASSWORD" :"***",
	"key":"***",
	"Key": "***",
	"key" : "***","key" :"***",
	"test_key":"***",
	"test_key": "***",
	"test_key" : "***","test_key" :"***",
	"KEY":"***",
	"KEY": "***",
	"KEY" : "***","KEY" :"***",
	"TEST_KEY":"***",
	"TEST_KEY": "***",
	"TEST_KEY" : "***","TEST_KEY" :"***",
	"ApiKey":"***",
	"ApiKey": "***",
	"ApiKey" : "***","ApiKey" :"***",
	"Token" : "***",
	"OtherField" : "OtherValue",
    "Visa" : "***",
    "MasterCard" : "***",
    "American Express" : "***",
    "Diners Club" : "***",
    "Discover" : "***",
    "JCB" : "***",
	"Security_Key": "***",
	"Secure_Data": "***",
	"Auth_Token": "***",
	"BearerToken": "***",
	"CertChain": "***",
	"CookieData": "***",
	"CredValue": "***",
	"DSN": "***",
	"JWT_Token": "***",
	"LoginUser": "***",
	"PwdHint": "***",
	"SealCode": "***",
	"SecretValue": "***",
	"SessionId": "***",
	"SgnPayload": "***",
	"SidValue": "***",
	"SigValue": "***",
	"Checksum": "***",
	"DSA_Signature": "***",
	"ECDSA_Signature": "***",
	"Fingerprint": "***",
	"HashValue": "***",
	"HmacValue": "***",
	"PKCS12": "***",
	"ProofData": "***",
	"RSA_Key": "***",
	"SaltValue": "***",
	"Attestation": "***",
	"Autograph": "***",
	"EndorseToken": "***",
	"AccNumber": "***",
	"AmountDue": "***",
	"BalTotal": "***",
	"BillRef": "***",
	"CardHolder": "***",
	"CC_Number": "***",
	"CV2_Code": "***",
	"CVC_Code": "***",
	"CVV_Code": "***",
	"IBAN_Code": "***",
	"PayRef": "***",
	"SwiftCode": "***",
	"AddrLine": "***",
	"BirthDate": "***",
	"CellNumber": "***",
	"DobValue": "***",
	"EmailAddr": "***",
	"ExpiryDate": "***",
	"HolderName": "***",
	"MailBox": "***",
	"PhoneNumber": "***",
	"PostalCode": "***",
	"SocialSecurity": "***",
	"SSN_Code": "***",
	"TaxId": "***",
	"TelNumber": "***",
	"First Name": "***",
	"Last-Name": "***",
	"Reference": "VISIBLE"
}
`
	got := HTTPData(data)
	require.Equal(t, expectedRedaction(expected), got)
}

func TestHTTPDataBytes(t *testing.T) {
	t.Parallel()

	input := []byte("Authorization: Bearer SECRET\napiKey=SECRET&reference=VISIBLE\n{\"password\":\"SECRET\",\"reference\":\"VISIBLE\"}")
	want := []byte(expectedRedaction("Authorization: ***\napiKey=***&reference=VISIBLE\n{\"password\":\"***\",\"reference\":\"VISIBLE\"}"))

	got := HTTPDataBytes(input)
	require.Equal(t, want, got)
}

func TestHTTPDataBytesMatchesHTTPData(t *testing.T) {
	t.Parallel()

	input := benchmarkHTTPDataInput
	require.Equal(t, HTTPData(input), string(HTTPDataBytes([]byte(input))))
}

func TestHTTPDataString(t *testing.T) {
	t.Parallel()

	input := []byte("Authorization: Bearer SECRET\npassword=SECRET&reference=VISIBLE")
	want := "Authorization: ***\npassword=***&reference=VISIBLE"

	got := HTTPDataString(input)
	require.Equal(t, want, got)
}

func TestHTTPDataBytesInto(t *testing.T) {
	t.Parallel()

	input := []byte("Authorization: Bearer SECRET\napiKey=SECRET&reference=VISIBLE\n{\"password\":\"SECRET\",\"reference\":\"VISIBLE\"}")
	want := []byte(expectedRedaction("Authorization: ***\napiKey=***&reference=VISIBLE\n{\"password\":\"***\",\"reference\":\"VISIBLE\"}"))

	dst := make([]byte, 0, len(input))
	got := HTTPDataBytesInto(dst, input)
	require.Equal(t, want, got)
}

func TestHTTPDataBytesIntoResetsDestination(t *testing.T) {
	t.Parallel()

	input := []byte("password=SECRET")
	dst := []byte("prefix should be overwritten")

	got := HTTPDataBytesInto(dst, input)
	require.Equal(t, []byte("password=***"), got)
}

func TestHTTPDataBytesPooled(t *testing.T) {
	t.Parallel()

	input := []byte("Authorization: Bearer SECRET\napiKey=SECRET&reference=VISIBLE")
	want := []byte("Authorization: ***\napiKey=***&reference=VISIBLE")

	var got []byte

	HTTPDataBytesPooled(input, func(out []byte) {
		got = append([]byte(nil), out...)
	})

	require.Equal(t, want, got)
}

func TestHTTPDataBytesPooledNilConsumer(t *testing.T) {
	t.Parallel()

	require.NotPanics(t, func() {
		HTTPDataBytesPooled([]byte("password=SECRET"), nil)
	})
}

func TestHTTPDataJSONNumericValues(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  string
	}{
		{`{"amount": 9999}`, `{"amount": "***"}`},
		{`{"cvv": true}`, `{"cvv": "***"}`},
		{`{"ssn": null}`, `{"ssn": "***"}`},
		{`{"balance": -1.5e3}`, `{"balance": "***"}`},
		{`{"password": 0}`, `{"password": "***"}`},
	}

	for _, tc := range cases {
		require.Equal(t, expectedRedaction(tc.want), HTTPData(tc.input), "input: %s", tc.input)
	}
}

func TestHTTPDataCreditCardWordBoundary(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  string
	}{
		// standalone card number (no surrounding punctuation)
		{"4012888888881881", "***"},
		// card adjacent to spaces
		{"card 4012888888881881 end", "card *** end"},
		// card at end of line followed by newline
		{"ref: 371449635398431\n", "ref: ***\n"},
		// card in parentheses (old behavior preserved)
		{"(4222222222222)", "(***)"},
	}

	for _, tc := range cases {
		require.Equal(t, expectedRedaction(tc.want), HTTPData(tc.input), "input: %s", tc.input)
	}
}

func TestHTTPDataURLEncodedNoFalsePositive(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  string
	}{
		// keyword in URL path must not cause a later param to be redacted
		{"GET /api/payment/receipt?reference=VISIBLE", "GET /api/payment/receipt?reference=VISIBLE"},
		// keyword in query param key should still be redacted
		{"GET /api/v1/status?session_id=SECRET", "GET /api/v1/status?session_id=***"},
		// keyword in path segment but innocent query param remains visible
		{"/authenticate?next=VISIBLE", "/authenticate?next=VISIBLE"},
	}

	for _, tc := range cases {
		require.Equal(t, expectedRedaction(tc.want), HTTPData(tc.input), "input: %s", tc.input)
	}
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

func TestRedactJSONKeysEdgeBranches(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "unmatched quote",
			input: `prefix "password`,
			want:  `prefix "password`,
		},
		{
			name:  "quoted token without colon",
			input: `"password" x`,
			want:  `"password" x`,
		},
		{
			name:  "colon then only whitespace",
			input: `{"password":   `,
			want:  `{"password":   `,
		},
		{
			name:  "string value with escapes",
			input: `{"password":"va\\\"l"}`,
			want:  `{"password":"***"}`,
		},
		{
			name:  "false literal redaction",
			input: `{"cvv":false}`,
			want:  `{"cvv":"***"}`,
		},
		{
			name:  "unknown value type",
			input: `{"password":[]}`,
			want:  `{"password":[]}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedRedaction(tc.want), string(redactJSONKeys([]byte(tc.input))))
		})
	}
}

func TestRedactURLEncodedKeysSlashInKey(t *testing.T) {
	t.Parallel()

	input := []byte("GET /x/password=SECRET&password=SECRET")
	want := []byte("GET /x/password=SECRET&password=***")
	require.Equal(t, want, redactURLEncodedKeys(input))
}

func TestRedactJSONKeysNonSensitivePreserved(t *testing.T) {
	t.Parallel()

	input := []byte(`{"reference":"VISIBLE"}`)
	want := []byte(`{"reference":"VISIBLE"}`)
	require.Equal(t, want, redactJSONKeys(input))
}

func TestRedactURLEncodedKeysNoEquals(t *testing.T) {
	t.Parallel()

	input := []byte("GET /health HTTP/1.1")
	want := []byte("GET /health HTTP/1.1")
	require.Equal(t, want, redactURLEncodedKeys(input))
}

func TestRedactURLEncodedKeysNonSensitivePreserved(t *testing.T) {
	t.Parallel()

	input := []byte("reference=VISIBLE&note=PUBLIC")
	want := []byte("reference=VISIBLE&note=PUBLIC")
	require.Equal(t, want, redactURLEncodedKeys(input))
}

func TestRedactionHelpersCoverageBranches(t *testing.T) {
	t.Parallel()

	t.Run("trailing newline absent", func(t *testing.T) {
		t.Parallel()
		require.Nil(t, trailingNewline([]byte("Authorization: Bearer SECRET")))
	})

	t.Run("authorization line without colon", func(t *testing.T) {
		t.Parallel()

		_, ok := redactAuthorizationLine([]byte("Authorization no-colon\n"))
		require.False(t, ok)
	})

	t.Run("json value start no colon after key", func(t *testing.T) {
		t.Parallel()

		_, hasKV, done := findJSONValueStart([]byte(`"password" x`), len(`"password"`)-1)
		require.False(t, hasKV)
		require.False(t, done)
	})

	t.Run("json value start end of input", func(t *testing.T) {
		t.Parallel()

		_, hasKV, done := findJSONValueStart([]byte(`"password"`), len(`"password"`)-1)
		require.False(t, hasKV)
		require.True(t, done)
	})

	t.Run("json string parser with trailing backslash", func(t *testing.T) {
		t.Parallel()

		src := []byte{0x22, 'a', 0x5c, 0x22}

		// Unterminated escaped sequence should return end-of-input.
		require.Equal(t, len(src), parseJSONStringEnd(src, 0))
	})
}

func TestIsSensitiveNormalizedKeyEmpty(t *testing.T) {
	t.Parallel()

	require.False(t, isSensitiveNormalizedKey(""))
}

func TestRedactCreditCardsKeepsDigitsAdjacentToWordChar(t *testing.T) {
	t.Parallel()

	input := []byte("prefix 4012888888881881x suffix")
	got := redactCreditCards(input)

	require.Equal(t, input, got)
}

func TestMatchesCardPatternReturnsFalseForUnknownPrefix(t *testing.T) {
	t.Parallel()

	require.False(t, matchesCardPattern([]byte("9111111111111111")))
}

func TestRedactAuthorizationLineBranches(t *testing.T) {
	t.Parallel()

	_, ok := redactAuthorizationLine([]byte("X-Header: SECRET\n"))
	require.False(t, ok)

	out, ok := redactAuthorizationLine([]byte("Authorization: Bearer SECRET\n"))
	require.True(t, ok)
	require.Equal(t, []byte("Authorization: ***\n"), out)
}

func TestTrailingNewlinePresent(t *testing.T) {
	t.Parallel()

	require.Equal(t, []byte("\n"), trailingNewline([]byte("line\n")))
}

func TestRedactCreditCardsAdditionalBranches(t *testing.T) {
	t.Parallel()

	// Match and redact a standalone valid card number.
	require.Equal(t, []byte("***"), redactCreditCards([]byte("4012888888881881")))

	// Keep non-matching numeric run unchanged.
	require.Equal(t, []byte("9111111111111111"), redactCreditCards([]byte("9111111111111111")))

	// Exercise branch where current digit follows a word character.
	require.Equal(t, []byte("x123"), redactCreditCards([]byte("x123")))
}

func TestSinglePassDigitWordBoundaryBranch(t *testing.T) {
	t.Parallel()

	input := []byte("(123x)")
	require.Equal(t, input, redactAllInSinglePass(input))
}

func TestAppendRedactedAuthorizationLineNoColon(t *testing.T) {
	t.Parallel()

	out, ok := appendRedactedAuthorizationLine([]byte("prefix "), []byte("Authorization no-colon\n"))
	require.False(t, ok)
	require.Equal(t, []byte("prefix "), out)
}

func TestNextLineBytes(t *testing.T) {
	t.Parallel()

	line, rest := nextLineBytes([]byte("a\nb"))
	require.Equal(t, []byte("a\n"), line)
	require.Equal(t, []byte("b"), rest)

	line, rest = nextLineBytes([]byte("abc"))
	require.Equal(t, []byte("abc"), line)
	require.Nil(t, rest)
}

func TestLikelyJSONKeyStart(t *testing.T) {
	t.Parallel()

	require.True(t, likelyJSONKeyStart([]byte(`"k":1`), 0))
	require.True(t, likelyJSONKeyStart([]byte(`{"k":1}`), 1))
	require.True(t, likelyJSONKeyStart([]byte(`{"a":1, "b":2}`), 8))
	require.False(t, likelyJSONKeyStart([]byte(`{"a":"v"}`), 6))
}

func TestFindJSONStringClosingQuote(t *testing.T) {
	t.Parallel()

	q, ok := findJSONStringClosingQuote([]byte(`a\"b"x`), 0)
	require.True(t, ok)
	require.Equal(t, 4, q)

	_, ok = findJSONStringClosingQuote([]byte(`a\"b`), 0)
	require.False(t, ok)
}

func TestAppendRedactedSensitiveJSONAtNoClosingQuote(t *testing.T) {
	t.Parallel()

	_, _, ok := appendRedactedSensitiveJSONAt([]byte(`"password`), 0, nil)
	require.False(t, ok)

	_, _, ok = appendRedactedSensitiveJSONAt([]byte(`"password"`), 0, nil)
	require.False(t, ok)
}

//nolint:paralleltest // Uses a shared sync.Pool to validate pooled buffer behavior.
func TestPooledBufferHelpers(t *testing.T) {
	// Intentionally not parallel: this test manipulates the shared pool.
	b := getPooledRedactionBuffer(2 << 20)
	require.GreaterOrEqual(t, cap(b), 2<<20)

	// Exercise defensive fallback when the pool contains an unexpected value type.
	redactionBufferPool.Put(&struct{}{})

	b = getPooledRedactionBuffer(64)
	require.GreaterOrEqual(t, cap(b), 64)

	putPooledRedactionBuffer(make([]byte, 0, 2<<20))
	putPooledRedactionBuffer(make([]byte, 0, 128))
}
