package random_test

import (
	"fmt"
	"log"

	"github.com/Vonage/gosrvlib/pkg/random"
)

//nolint:testableexamples
func ExampleRnd_RandUint32() {
	r := random.New(nil)

	n := r.RandUint32()

	fmt.Println(n)
}

//nolint:testableexamples
func ExampleRnd_RandUint64() {
	r := random.New(nil)

	n := r.RandUint64()

	fmt.Println(n)
}

//nolint:testableexamples
func ExampleRnd_RandString() {
	r := random.New(nil)

	s, err := r.RandString(16)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(s)
}
