package encode_test

import (
	"fmt"
	"log"

	"github.com/Vonage/gosrvlib/pkg/encode"
)

//nolint:testableexamples
func ExampleEncode() {
	type TestData struct {
		Alpha string
		Beta  int
	}

	data := &TestData{Alpha: "test_string", Beta: -9876}

	v, err := encode.Encode(data)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(v)
}

func ExampleDecode() {
	type TestData struct {
		Alpha string
		Beta  int
	}

	var data TestData

	msg := "Kf+BAwEBCFRlc3REYXRhAf+CAAECAQVBbHBoYQEMAAEEQmV0YQEEAAAAD/+CAQZhYmMxMjMB/gLtAA=="

	err := encode.Decode(msg, &data)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(data)

	// Output:
	// {abc123 -375}
}

func ExampleSerialize() {
	type TestData struct {
		Alpha string
		Beta  int
	}

	data := &TestData{Alpha: "test_string", Beta: -9876}

	v, err := encode.Serialize(data)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(v)

	// Output:
	// eyJBbHBoYSI6InRlc3Rfc3RyaW5nIiwiQmV0YSI6LTk4NzZ9Cg==
}

func ExampleDeserialize() {
	type TestData struct {
		Alpha string
		Beta  int
	}

	var data TestData

	msg := "eyJBbHBoYSI6ImFiYzEyMyIsIkJldGEiOi0zNzV9Cg=="

	err := encode.Deserialize(msg, &data)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(data)

	// Output:
	// {abc123 -375}
}
