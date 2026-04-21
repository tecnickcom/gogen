package random_test

import (
	"fmt"

	"github.com/tecnickcom/gogen/pkg/random"
)

//nolint:testableexamples
func ExampleRnd_UUIDv7() {
	r := random.New(nil)
	v := r.UUIDv7()

	fmt.Println(v)
}

//nolint:testableexamples
func ExampleUUID_Format() {
	r := random.New(nil)
	u := r.UUIDv7()

	var b [36]byte

	u.Format(&b)

	fmt.Println(string(b[:]))
}

//nolint:testableexamples
func ExampleUUID_Byte() {
	r := random.New(nil)
	v := r.UUIDv7().Byte()

	fmt.Println(string(v))
}

//nolint:testableexamples
func ExampleUUID_String() {
	r := random.New(nil)
	v := r.UUIDv7().String()

	fmt.Println(v)
}
