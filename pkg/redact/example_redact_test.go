package redact_test

import (
	"fmt"

	"github.com/tecnickcom/gogen/pkg/redact"
)

func ExampleHTTPData() {
	// example input data
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
alpha=beta&password=SECRET&key=SECRET&gamma=delta
Token=SECRET

{
	"password": "SECRET",
	"test_password": "SECRET",
	"PASSWORD": "SECRET",
	"TEST_PASSWORD": "SECRET",
	"key": "SECRET",
	"test_key": "SECRET",
	"KEY": "SECRET",
	"TEST_KEY": "SECRET",
	"Token": "SECRET",
	"Visa" : "4012888888881881",
	"MasterCard" : "5555555555554444",
	"American Express" : "371449635398431",
	"Diners Club" : "38520000023237",
	"Discover" : "6011000990139424",
	"JCB" : "3566002020360505"
}
`
	// redact input data
	redactedData := redact.HTTPData(data)

	fmt.Println(redactedData)

	// Output:
	// GET /v1/version HTTP/1.1
	// Host: test.redact.invalid
	// User-Agent: Go-http-client/1.1
	// Authorization: @~REDACTED~@
	// authorization : @~REDACTED~@
	// X-GOGEN-Trace-Id: abcdef0123456789
	// Accept-Encoding: gzip
	//
	// password=@~REDACTED~@
	// test_password=@~REDACTED~@
	// PASSWORD=@~REDACTED~@
	// TEST_PASSWORD=@~REDACTED~@
	// key=@~REDACTED~@
	// test_key=@~REDACTED~@
	// KEY=@~REDACTED~@
	// TEST_KEY=@~REDACTED~@
	// password=@~REDACTED~@&key=@~REDACTED~@
	// alpha=beta&password=@~REDACTED~@&key=@~REDACTED~@&gamma=delta
	// Token=@~REDACTED~@
	//
	// {
	// 	"password": "@~REDACTED~@",
	// 	"test_password": "@~REDACTED~@",
	// 	"PASSWORD": "@~REDACTED~@",
	// 	"TEST_PASSWORD": "@~REDACTED~@",
	// 	"key": "@~REDACTED~@",
	// 	"test_key": "@~REDACTED~@",
	// 	"KEY": "@~REDACTED~@",
	// 	"TEST_KEY": "@~REDACTED~@",
	// 	"Token": "@~REDACTED~@",
	// 	"Visa" : "@~REDACTED~@",
	// 	"MasterCard" : "@~REDACTED~@",
	// 	"American Express" : "@~REDACTED~@",
	// 	"Diners Club" : "@~REDACTED~@",
	// 	"Discover" : "@~REDACTED~@",
	// 	"JCB" : "@~REDACTED~@"
	// }
}
