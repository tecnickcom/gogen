package redact_test

import (
	"fmt"

	"github.com/tecnickcom/gogen/pkg/redact"
)

const (
	testData = `
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
)

func ExampleHTTPData() {
	// redact input data
	redactedData := redact.HTTPData(testData)

	fmt.Println(redactedData)

	// Output:
	// GET /v1/version HTTP/1.1
	// Host: test.redact.invalid
	// User-Agent: Go-http-client/1.1
	// Authorization: ***
	// authorization : ***
	// X-GOGEN-Trace-Id: abcdef0123456789
	// Accept-Encoding: gzip
	//
	// password=***
	// test_password=***
	// PASSWORD=***
	// TEST_PASSWORD=***
	// key=***
	// test_key=***
	// KEY=***
	// TEST_KEY=***
	// password=***&key=***
	// alpha=beta&password=***&key=***&gamma=delta
	// Token=***
	//
	// {
	// 	"password": "***",
	// 	"test_password": "***",
	// 	"PASSWORD": "***",
	// 	"TEST_PASSWORD": "***",
	// 	"key": "***",
	// 	"test_key": "***",
	// 	"KEY": "***",
	// 	"TEST_KEY": "***",
	// 	"Token": "***",
	// 	"Visa" : "***",
	// 	"MasterCard" : "***",
	// 	"American Express" : "***",
	// 	"Diners Club" : "***",
	// 	"Discover" : "***",
	// 	"JCB" : "***"
	// }
}

func ExampleHTTPDataString() {
	// redact input data
	redactedData := redact.HTTPDataString([]byte(testData))

	fmt.Println(redactedData)

	// Output:
	// GET /v1/version HTTP/1.1
	// Host: test.redact.invalid
	// User-Agent: Go-http-client/1.1
	// Authorization: ***
	// authorization : ***
	// X-GOGEN-Trace-Id: abcdef0123456789
	// Accept-Encoding: gzip
	//
	// password=***
	// test_password=***
	// PASSWORD=***
	// TEST_PASSWORD=***
	// key=***
	// test_key=***
	// KEY=***
	// TEST_KEY=***
	// password=***&key=***
	// alpha=beta&password=***&key=***&gamma=delta
	// Token=***
	//
	// {
	// 	"password": "***",
	// 	"test_password": "***",
	// 	"PASSWORD": "***",
	// 	"TEST_PASSWORD": "***",
	// 	"key": "***",
	// 	"test_key": "***",
	// 	"KEY": "***",
	// 	"TEST_KEY": "***",
	// 	"Token": "***",
	// 	"Visa" : "***",
	// 	"MasterCard" : "***",
	// 	"American Express" : "***",
	// 	"Diners Club" : "***",
	// 	"Discover" : "***",
	// 	"JCB" : "***"
	// }
}

func ExampleHTTPDataBytesInto() {
	inputs := [][]byte{
		[]byte("password=SECRET&reference=VISIBLE"),
		[]byte("token=SECRET&note=PUBLIC"),
	}

	var dst []byte

	for _, in := range inputs {
		dst = redact.HTTPDataBytesInto(dst, in)
		fmt.Println(string(dst))
	}

	// Output:
	// password=***&reference=VISIBLE
	// token=***&note=PUBLIC
}
