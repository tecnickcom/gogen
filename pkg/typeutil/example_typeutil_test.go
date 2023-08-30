package typeutil_test

import (
	"fmt"

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

func ExampleZero() {
	num := 5

	v := typeutil.Zero(num)
	fmt.Println(v)

	// Output:
	// 0
}

//nolint:testableexamples
func ExamplePointer() {
	v := 5

	p := typeutil.Pointer(v)
	fmt.Println(p)
}

func ExampleValue() {
	num := 5
	p := &num

	v := typeutil.Value(p)
	fmt.Println(v)

	// Output:
	// 5
}
