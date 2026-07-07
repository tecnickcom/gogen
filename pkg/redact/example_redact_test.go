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

func ExampleString() {
	// String is the canonical string-in/string-out entry point.
	fmt.Println(redact.String("X-Api-Key: SECRET\npassword=SECRET&note=ok"))

	// Output:
	// X-Api-Key: ***
	// password=***&note=ok
}

func ExampleAppendTo() {
	inputs := [][]byte{
		[]byte("token=SECRET&reference=VISIBLE"),
		[]byte(`{"card":"4012 8888 8888 1881"}`),
	}

	// AppendTo reuses the destination buffer across calls to avoid
	// per-call allocations on high-throughput logging paths.
	var dst []byte

	for _, in := range inputs {
		dst = redact.AppendTo(dst, in)
		fmt.Println(string(dst))
	}

	// Output:
	// token=***&reference=VISIBLE
	// {"card":"***"}
}

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

func ExampleNew() {
	// A custom Redactor instance: different marker, an extra sensitive key
	// token, and financial fields kept readable.
	re := redact.New(
		redact.WithMarker("#REDACTED#"),
		redact.WithExtraTokens("floof"),
		redact.WithoutTokens("amount"),
	)

	fmt.Println(re.String(`{"floof":"secret","amount":42,"password":"x"}`))

	// Output:
	// {"floof":"#REDACTED#","amount":42,"password":"#REDACTED#"}
}
