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
X-Nexmo-Trace-Id: abcdef0123456789
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
alpha=beta&password=SECRET&key=SECRET&gamma=delta

{
	"password":"SECRET",
	"password": "SECRET",
	"password" : "SECRET",
	"password" :"SECRET",
	"test_password":"SECRET",
	"test_password": "SECRET",
	"test_password" : "SECRET",
	"test_password" :"SECRET",
	"PASSWORD":"SECRET",
	"PASSWORD": "SECRET",
	"PASSWORD" : "SECRET",
	"PASSWORD" :"SECRET",
	"TEST_PASSWORD":"SECRET",
	"TEST_PASSWORD": "SECRET",
	"TEST_PASSWORD" : "SECRET",
	"TEST_PASSWORD" :"SECRET",
	"key":"SECRET",
	"key": "SECRET",
	"key" : "SECRET",
	"key" :"SECRET",
	"test_key":"SECRET",
	"test_key": "SECRET",
	"test_key" : "SECRET",
	"test_key" :"SECRET",
	"KEY":"SECRET",
	"KEY": "SECRET",
	"KEY" : "SECRET",
	"KEY" :"SECRET",
	"TEST_KEY":"SECRET",
	"TEST_KEY": "SECRET",
	"TEST_KEY" : "SECRET",
	"TEST_KEY" :"SECRET"
}
`
	expected := `
GET /v1/version HTTP/1.1
Host: test.redact.invalid
User-Agent: Go-http-client/1.1
Authorization: @~REDACTED~@
authorization : @~REDACTED~@
X-Nexmo-Trace-Id: abcdef0123456789
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
alpha=beta&password=@~REDACTED~@&key=@~REDACTED~@&gamma=delta

{
	"password":"@~REDACTED~@",
	"password": "@~REDACTED~@",
	"password" : "@~REDACTED~@",
	"password" :"@~REDACTED~@",
	"test_password":"@~REDACTED~@",
	"test_password": "@~REDACTED~@",
	"test_password" : "@~REDACTED~@",
	"test_password" :"@~REDACTED~@",
	"PASSWORD":"@~REDACTED~@",
	"PASSWORD": "@~REDACTED~@",
	"PASSWORD" : "@~REDACTED~@",
	"PASSWORD" :"@~REDACTED~@",
	"TEST_PASSWORD":"@~REDACTED~@",
	"TEST_PASSWORD": "@~REDACTED~@",
	"TEST_PASSWORD" : "@~REDACTED~@",
	"TEST_PASSWORD" :"@~REDACTED~@",
	"key":"@~REDACTED~@",
	"key": "@~REDACTED~@",
	"key" : "@~REDACTED~@",
	"key" :"@~REDACTED~@",
	"test_key":"@~REDACTED~@",
	"test_key": "@~REDACTED~@",
	"test_key" : "@~REDACTED~@",
	"test_key" :"@~REDACTED~@",
	"KEY":"@~REDACTED~@",
	"KEY": "@~REDACTED~@",
	"KEY" : "@~REDACTED~@",
	"KEY" :"@~REDACTED~@",
	"TEST_KEY":"@~REDACTED~@",
	"TEST_KEY": "@~REDACTED~@",
	"TEST_KEY" : "@~REDACTED~@",
	"TEST_KEY" :"@~REDACTED~@"
}
`
	got := HTTPData(data)
	require.Equal(t, expected, got)
}
