package decint_test

import (
	"fmt"
	"log"

	"github.com/vonage/gosrvlib/pkg/decint"
)

func ExampleFloatToInt() {
	v := decint.FloatToInt(123.456)

	fmt.Println(v)

	// Output:
	// 123456000
}

func ExampleIntToFloat() {
	v := decint.IntToFloat(123456)

	fmt.Println(v)

	// Output:
	// 0.123456
}

func ExampleStringToInt() {
	v, err := decint.StringToInt("123456")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(v)

	// Output:
	// 123456000000
}

func ExampleIntToString() {
	v := decint.IntToString(123456)

	fmt.Println(v)

	// Output:
	// 0.123456
}
