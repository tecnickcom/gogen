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
func ExampleRnd_UID128() {
	r := random.New(nil)
	v := r.UID128()

	fmt.Println(v)
}
