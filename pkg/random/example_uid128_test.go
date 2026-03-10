package random_test

import (
	"fmt"

	"github.com/tecnickcom/gogen/pkg/random"
)

//nolint:testableexamples
func ExampleRnd_UID128() {
	r := random.New(nil)
	v := r.UID128()

	fmt.Println(v)
}

//nolint:testableexamples
func ExampleTUID128_Hex() {
	r := random.New(nil)
	v := r.UID128().Hex()

	fmt.Println(v)
}

//nolint:testableexamples
func ExampleTUID128_String() {
	r := random.New(nil)
	v := r.UID128().String()

	fmt.Println(v)
}
