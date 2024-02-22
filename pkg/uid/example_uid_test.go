package uid_test

import (
	"fmt"

	"github.com/Vonage/gosrvlib/pkg/uid"
)

//nolint:testableexamples
func ExampleNewID64() {
	v := uid.NewID64()

	fmt.Println(v)
}

//nolint:testableexamples
func ExampleNewID128() {
	v := uid.NewID128()

	fmt.Println(v)
}
