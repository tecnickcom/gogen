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
