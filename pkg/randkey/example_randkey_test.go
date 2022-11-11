package randkey_test

import (
	"fmt"

	"github.com/nexmoinc/gosrvlib/pkg/randkey"
)

//nolint:testableexamples
func ExampleNew() {
	// generate a new key
	k := randkey.New()

	fmt.Println(k)
}

//nolint:testableexamples
func ExampleRandKey_Key() {
	// generate a new random key as uint64
	k := randkey.New().Key()

	fmt.Println(k)
}

//nolint:testableexamples
func ExampleRandKey_String() {
	// generate a new random key as 36-char encoded string
	k := randkey.New().String()

	fmt.Println(k)
}

//nolint:testableexamples
func ExampleRandKey_Hex() {
	// generate a new random key as fixed-length 16 digits hexadecimal string key.
	k := randkey.New().Hex()

	fmt.Println(k)
}
