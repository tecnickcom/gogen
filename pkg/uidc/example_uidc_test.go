package uidc_test

import (
	"fmt"

	"github.com/nexmoinc/gosrvlib/pkg/uidc"
)

//nolint:testableexamples
func ExampleNewID64() {
	v := uidc.NewID64()

	fmt.Println(v)
}

//nolint:testableexamples
func ExampleNewID128() {
	v := uidc.NewID128()

	fmt.Println(v)
}
