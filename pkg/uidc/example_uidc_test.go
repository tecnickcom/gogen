package uidc_test

import (
	"fmt"

	"github.com/nexmoinc/gosrvlib/pkg/uidc"
)

func ExampleNewID64() {
	v := uidc.NewID64()

	fmt.Println(v)
}

func ExampleNewID128() {
	v := uidc.NewID128()

	fmt.Println(v)
}
