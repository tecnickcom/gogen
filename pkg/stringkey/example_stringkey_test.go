package stringkey_test

import (
	"fmt"

	"github.com/nexmoinc/gosrvlib/pkg/stringkey"
)

func ExampleNew() {
	// input strings
	args := []string{
		"0123456789",
		"abcdefghijklmnopqrstuvwxyz",
		"Lorem ipsum dolor sit amet",
	}

	// generate a new key
	sk := stringkey.New(args...)

	fmt.Println(sk)

	// Output:
	// 2p8dmari397l8
}

func ExampleStringKey_Key() {
	// generate a new key as uint64
	k := stringkey.New(
		"0123456789",
		"abcdefghijklmnopqrstuvwxyz",
		"Lorem ipsum dolor sit amet",
	).Key()

	fmt.Println(k)

	// Output:
	// 12797937727583693228
}

func ExampleStringKey_String() {
	// generate a new key as 36-char encoded string
	k := stringkey.New(
		"0123456789",
		"abcdefghijklmnopqrstuvwxyz",
		"Lorem ipsum dolor sit amet",
	).String()

	fmt.Println(k)

	// Output:
	// 2p8dmari397l8
}

func ExampleStringKey_Hex() {
	// generate a new key as fixed-length 16 digits hexadecimal string key.
	k := stringkey.New(
		"0123456789",
		"abcdefghijklmnopqrstuvwxyz",
		"Lorem ipsum dolor sit amet",
	).Hex()

	fmt.Println(k)

	// Output:
	// b19b688e8e3229ac
}
