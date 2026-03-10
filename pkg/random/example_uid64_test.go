package random_test

import (
	"fmt"

	"github.com/tecnickcom/gogen/pkg/random"
)

//nolint:testableexamples
func ExampleRnd_UID64() {
	r := random.New(nil)
	v := r.UID64()

	fmt.Println(v)
}

//nolint:testableexamples
func ExampleTUID64_Hex() {
	r := random.New(nil)
	v := r.UID64().Hex()

	fmt.Println(v)
}

//nolint:testableexamples
func ExampleTUID64_String() {
	r := random.New(nil)
	v := r.UID64().String()

	fmt.Println(v)
}
