package redact

import (
	"strings"
	"sync"
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
X-NURAGO-Trace-Id: abcdef0123456789
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
secure_data=VISIBLE
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
addr_line=VISIBLE
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
	"Secure_Data": "VISIBLE",
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
	"AddrLine": "VISIBLE",
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
X-NURAGO-Trace-Id: abcdef0123456789
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
secure_data=VISIBLE
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
addr_line=VISIBLE
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
MasterCard2Series=***
MasterCard5Series=***
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
	"Secure_Data": "VISIBLE",
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
	"AddrLine": "VISIBLE",
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

//nolint:paralleltest // Swaps the shared pool to validate pooled buffer behavior.
func TestPooledBufferHelpers(t *testing.T) {
	// Intentionally not parallel: this test replaces the shared pool. Restore an
	// equivalent pool afterwards so other tests/benchmarks keep working.
	t.Cleanup(func() { redactionBufferPool = sync.Pool{New: newRedactionBuffer} })

	// Pool whose New returns a too-small buffer, forcing the grow-path fallback.
	redactionBufferPool = sync.Pool{New: func() any {
		b := make([]byte, 0, 1)

		return &b
	}}

	b := getPooledRedactionBuffer(2 << 20)
	require.GreaterOrEqual(t, cap(b), 2<<20)

	// Pool whose New returns an unexpected value type, forcing the nil-assertion
	// fallback in getPooledRedactionBuffer.
	redactionBufferPool = sync.Pool{New: func() any { return &struct{}{} }}

	b = getPooledRedactionBuffer(64)
	require.GreaterOrEqual(t, cap(b), 64)

	// Pool whose New returns a usable buffer, exercising the reuse path.
	redactionBufferPool = sync.Pool{New: func() any {
		b := make([]byte, 0, 256)

		return &b
	}}

	b = getPooledRedactionBuffer(64)
	require.GreaterOrEqual(t, cap(b), 64)

	// Oversized buffers are dropped; right-sized buffers are returned to the pool.
	putPooledRedactionBuffer(make([]byte, 0, 2<<20))
	putPooledRedactionBuffer(make([]byte, 0, 128))
}

func TestCanonicalAPI(t *testing.T) {
	t.Parallel()

	input := benchmarkHTTPDataInput
	want := String(input) // canonical reference; every other entry point must agree

	require.Equal(t, want, HTTPData(input)) // the compatibility alias delegates to String
	require.Equal(t, want, string(Bytes([]byte(input))))
	require.Equal(t, want, BytesToString([]byte(input)))

	var dst []byte

	dst = AppendTo(dst, []byte(input))
	require.Equal(t, want, string(dst))

	var pooled string

	Pooled([]byte(input), func(out []byte) { pooled = string(out) })
	require.Equal(t, want, pooled)

	require.NotPanics(t, func() { Pooled([]byte(input), nil) })
}

// TestAppendToInPlaceAliasing verifies that an in-place AppendTo(b, b) — where
// the destination and source share storage — produces the same output as a
// clean redaction, instead of corrupting the buffer and leaking a secret whose
// key the write cursor overran.
func TestAppendToInPlaceAliasing(t *testing.T) {
	t.Parallel()

	inputs := []string{
		"pin=1&password=SUPERSECRET",
		"a=1&b=2&password=SUPERSECRET",
		"cvv=1&cvv=2&token=SUPERSECRET&x=9",
		benchmarkHTTPDataInput,
	}

	for _, in := range inputs {
		buf := make([]byte, 0, len(in)+16)
		buf = append(buf, in...)

		require.Equal(t, String(in), string(AppendTo(buf, buf)), "in-place alias for %q", in)
	}
}

// TestRedactionIdempotency locks the property that redacting already-redacted
// output is a no-op: every rule replaces values with the marker and the marker
// never re-triggers a rule. Layered redaction (e.g. middleware + log sink)
// must not double-mangle output.
func TestRedactionIdempotency(t *testing.T) {
	t.Parallel()

	docs := []string{
		benchmarkHTTPDataInput,
		"Authorization: Basic QQ==\r\nCookie: sid=1\r\n\r\n{\"password\":\"x\"}",
		`{"password":{"a":"b"},"card":"4012 8888 8888 1881"}`,
		"token=SECRET&note=ok\npassword=\"quoted secret\" tail",
		"X-Api-Key: k\npan 6759 6498 2643 8453 end",
		"dial error: postgres://app:hunter2@10.0.0.5/db?sslmode=disable",
		"state=eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxIn0.dQw4w9WgXcQ",
		"-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEA\n-----END RSA PRIVATE KEY-----\n",
		"<password>SECRET</password><note>ok</note>",
	}

	for _, doc := range docs {
		once := String(doc)
		require.Equal(t, once, String(once), "not idempotent for input: %q", doc)
	}
}

// FuzzRedactionIdempotency searches for inputs where redaction fails to reach
// a fixed point after one extra pass, and doubles as a no-panic robustness
// fuzz. Redaction is byte-stable in a single pass on well-formed input (the
// strict table tests pin that); on adversarial ambiguous quote soup a rewrite
// can change how the next pass parses the surroundings, so the guaranteed
// property is convergence: the second pass is a fixed point, and re-redaction
// only ever redacts more, never less. A violation here means stacked
// redaction layers (middleware + log sink) would keep mangling output.
//
// Both properties are checked, on the default redactor and on a configured one
// (custom marker, a disabled rule class, Luhn gate on): a fixed point alone is
// not enough, since an engine that un-redacts on the second pass and stabilizes
// on the third would still reach one.
func FuzzRedactionIdempotency(f *testing.F) {
	seeds := []string{
		benchmarkHTTPDataInput,
		"Authorization: Basic QQ==\r\nCookie: sid=1\r\n\r\n{\"password\":\"x\"}",
		`{"password":{"a":"b"},"card":"4012 8888 8888 1881"}`,
		"token=SECRET&note=ok\npassword=\"quoted secret\" tail",
		"pass=\"\"0", "password=***x", "a=\"***\"", "token=*** x",
		"dial error: postgres://app:hunter2@10.0.0.5/db", "://:@", "http://u:p@ss@h",
		"state=eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxIn0.sig", "eyJ.",
		"-----BEGIN RSA PRIVATE KEY-----\nMIIE\n-----END RSA PRIVATE KEY-----\n",
		"-----BEGIN RSA PRIVATE KEY-----\\nMIIE\\n-----END RSA PRIVATE KEY-----",
		"<password>SECRET</password>", "<a>***</a>", "<password><![CDATA[x]]></password>",
		"pan 6759 6498 2643 8453 end", "1 800 555 0199 1234",
		"ghp_AbCdEfGhIjKlMnOpQrStUvWxYz012345", "AKIAIOSFODNN7EXAMPLE",
		"PGPASSWORD=secret", "newpassword=x&oldpassword=y", "note=#R# tail",
		`{"body":"{\"password\":\"hunter2\"}"}`, `{\"password\":\"x`,
		`{"name":"DB_PASSWORD","value":"hunter2"}`, `{"key":"api_key","value":"AKIA"}`,
		"https://app/cb#access_token=SECRET&type=Bearer",
		"  password: hunter2\n", "-----BEGIN PGP PRIVATE KEY BLOCK-----\nMIIE\n-----END PGP PRIVATE KEY BLOCK-----\n",
		"root:hunter2@tcp(127.0.0.1:3306)/db",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	// Built once: New() pre-sizes a key memo, which is too costly per iteration.
	configured := New(WithMarker("#R#"), WithoutRules(RuleJSON), WithLuhnCheck(true))

	f.Fuzz(func(t *testing.T, s string) {
		assertConvergent(t, Default(), RedactionMarker, s)
		assertConvergent(t, configured, "#R#", s)
	})
}

// assertConvergent checks the two guaranteed properties of a redaction pass on
// one input: convergence (the second pass is a fixed point) and monotonicity
// (each pass redacts at least as much as the previous one, never less).
func assertConvergent(t *testing.T, re *Redactor, marker, s string) {
	t.Helper()

	once := re.String(s)
	twice := re.String(once)
	thrice := re.String(twice)

	if twice != thrice {
		t.Fatalf("redaction does not converge after two passes:\nin    : %q\nonce  : %q\ntwice : %q\nthrice: %q",
			s, once, twice, thrice)
	}

	// Monotonicity: a later pass can only add markers. Counting them is a
	// proxy for "reveals no more than before" that a re-redaction cannot fake:
	// dropping a marker means a value that was hidden became visible again.
	if got, want := strings.Count(twice, marker), strings.Count(once, marker); got < want {
		t.Fatalf("re-redaction removed markers (%d -> %d):\nin   : %q\nonce : %q\ntwice: %q",
			want, got, s, once, twice)
	}
}

// TestRedactionConvergence pins the known pathological inputs where a single
// pass is not byte-stable (ambiguous quote soup whose rewrite changes how the
// next pass parses the surroundings): the second pass must be a fixed point,
// and each extra pass only redacts more, never less.
func TestRedactionConvergence(t *testing.T) {
	t.Parallel()

	docs := []string{
		`sid=0"sid":0`,                   // non-key context replaced by marker
		"\"pass\":{\",\"Card\":\"{\"\"}", // container balance flips after inner rewrite
	}

	for _, doc := range docs {
		once := String(doc)
		twice := String(once)
		require.Equal(t, twice, String(twice), "no fixed point after two passes for input: %q", doc)
	}
}
