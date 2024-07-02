package stringmetric_test

import (
	"fmt"

	"github.com/Vonage/gosrvlib/pkg/stringmetric"
)

func ExampleDLDistance() {
	d := stringmetric.DLDistance("a cat", "a abct")

	// "a cat" (one transposition)-> "a act" (one insertion)-> "a abct"

	fmt.Println(d)

	// Output:
	// 2
}
