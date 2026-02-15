package redact

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHTTPData(t *testing.T) {
	t.Parallel()

	data := `
GET /v1/version HTTP/1.1
Host: test.redact.invalid
User-Agent: Go-http-client/1.1
Authorization: Basic SECRET_ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789=
authorization : ApiKey=SECRET OtherData=SECRET
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
    "JCB" : "3566002020360505"
}
`
	expected := `
GET /v1/version HTTP/1.1
Host: test.redact.invalid
User-Agent: Go-http-client/1.1
Authorization: @~REDACTED~@
authorization : @~REDACTED~@
X-GOGEN-Trace-Id: abcdef0123456789
Accept-Encoding: gzip

password=@~REDACTED~@
test_password=@~REDACTED~@
PASSWORD=@~REDACTED~@
TEST_PASSWORD=@~REDACTED~@
key=@~REDACTED~@
test_key=@~REDACTED~@
KEY=@~REDACTED~@
TEST_KEY=@~REDACTED~@
password=@~REDACTED~@&key=@~REDACTED~@
ApiKey=@~REDACTED~@&alpha=beta&password=@~REDACTED~@&key=@~REDACTED~@&gamma=delta
Token=@~REDACTED~@

{
	"password":"@~REDACTED~@",
	"Password": "@~REDACTED~@",
	"password" : "@~REDACTED~@","password" :"@~REDACTED~@",
	"test_password":"@~REDACTED~@",
	"test_password_test": "@~REDACTED~@",
	"test_password" : "@~REDACTED~@","test_password" :"@~REDACTED~@",
	"PASSWORD":"@~REDACTED~@",
	"PASSWORD": "@~REDACTED~@",
	"PASSWORD" : "@~REDACTED~@","PASSWORD" :"@~REDACTED~@",
	"TEST_PASSWORD":"@~REDACTED~@",
	"TEST_PASSWORD": "@~REDACTED~@",
	"TEST_PASSWORD" : "@~REDACTED~@","TEST_PASSWORD" :"@~REDACTED~@",
	"key":"@~REDACTED~@",
	"Key": "@~REDACTED~@",
	"key" : "@~REDACTED~@","key" :"@~REDACTED~@",
	"test_key":"@~REDACTED~@",
	"test_key": "@~REDACTED~@",
	"test_key" : "@~REDACTED~@","test_key" :"@~REDACTED~@",
	"KEY":"@~REDACTED~@",
	"KEY": "@~REDACTED~@",
	"KEY" : "@~REDACTED~@","KEY" :"@~REDACTED~@",
	"TEST_KEY":"@~REDACTED~@",
	"TEST_KEY": "@~REDACTED~@",
	"TEST_KEY" : "@~REDACTED~@","TEST_KEY" :"@~REDACTED~@",
	"ApiKey":"@~REDACTED~@",
	"ApiKey": "@~REDACTED~@",
	"ApiKey" : "@~REDACTED~@","ApiKey" :"@~REDACTED~@",
	"Token" : "@~REDACTED~@",
	"OtherField" : "OtherValue",
    "Visa" : "@~REDACTED~@",
    "MasterCard" : "@~REDACTED~@",
    "American Express" : "@~REDACTED~@",
    "Diners Club" : "@~REDACTED~@",
    "Discover" : "@~REDACTED~@",
    "JCB" : "@~REDACTED~@"
}
`
	got := HTTPData(data)
	require.Equal(t, expected, got)
}
