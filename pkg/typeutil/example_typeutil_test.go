package typeutil_test

import (
	"fmt"
	"log"

	"github.com/Vonage/gosrvlib/pkg/typeutil"
)

func ExampleIsNil() {
	var nilChan chan int

	v := typeutil.IsNil(nilChan)
	fmt.Println(v)

	// Output:
	// true
}

func ExampleIsZero() {
	var zeroInt int

	v := typeutil.IsZero(zeroInt)
	fmt.Println(v)

	// Output:
	// true
}

//nolint:testableexamples
func ExampleEncode() {
	type TestData struct {
		Alpha string
		Beta  int
	}

	data := &TestData{Alpha: "test_string", Beta: -9876}

	v, err := typeutil.Encode(data)
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

	err := typeutil.Decode(msg, &data)
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

	v, err := typeutil.Serialize(data)
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

	err := typeutil.Deserialize(msg, &data)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(data)

	// Output:
	// {abc123 -375}
}
