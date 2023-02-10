package decint_test

import (
	"fmt"
	"log"

	"github.com/vonage/gosrvlib/pkg/decint"
)

func ExampleFloatToUint() {
	v := decint.FloatToUint(123.456)

	fmt.Println(v)

	// Output:
	// 123456000
}

func ExampleUintToFloat() {
	v := decint.UintToFloat(123456)

	fmt.Println(v)

	// Output:
	// 0.123456
}

func ExampleStringToUint() {
	v, err := decint.StringToUint("123.456")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(v)

	// Output:
	// 123456000
}

func ExampleUintToString() {
	v := decint.UintToString(123456)

	fmt.Println(v)

	// Output:
	// 0.123456
}
