package uid_test

import (
	"fmt"
	"log"

	"github.com/nexmoinc/gosrvlib/pkg/uid"
)

func ExampleNewID64() {
	err := uid.InitRandSeed()
	if err != nil {
		log.Fatal(err)
	}

	v := uid.NewID64()

	fmt.Println(v)
}

func ExampleNewID128() {
	err := uid.InitRandSeed()
	if err != nil {
		log.Fatal(err)
	}

	v := uid.NewID128()

	fmt.Println(v)
}
