package uid_test

import (
	"fmt"
	"log"

	"github.com/Vonage/gosrvlib/pkg/uid"
)

//nolint:testableexamples
func ExampleNewID64() {
	err := uid.InitRandSeed()
	if err != nil {
		log.Fatal(err)
	}

	v := uid.NewID64()

	fmt.Println(v)
}

//nolint:testableexamples
func ExampleNewID128() {
	err := uid.InitRandSeed()
	if err != nil {
		log.Fatal(err)
	}

	v := uid.NewID128()

	fmt.Println(v)
}
