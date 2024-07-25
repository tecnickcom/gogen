package phonekeypad_test

import (
	"fmt"

	"github.com/Vonage/gosrvlib/pkg/phonekeypad"
)

func ExampleKeypadDigit() {
	phoneNumber := "999-EXAMPLE-1"
	numSeq := make([]int, 0, len(phoneNumber))

	for _, r := range phoneNumber {
		v, status := phonekeypad.KeypadDigit(r)
		if status {
			numSeq = append(numSeq, v)
		}
	}

	fmt.Println(numSeq)

	// Output:
	// [9 9 9 3 9 2 6 7 5 3 1]
}

func ExampleKeypadNumber() {
	phoneNumber := "999-EXAMPLE-2"

	numSeq := phonekeypad.KeypadNumber(phoneNumber)

	fmt.Println(numSeq)

	// Output:
	// [9 9 9 3 9 2 6 7 5 3 2]
}

func ExampleKeypadNumberString() {
	phoneNumber := "999-EXAMPLE-3"

	numStr := phonekeypad.KeypadNumberString(phoneNumber)

	fmt.Println(numStr)

	// Output:
	// 99939267533
}
