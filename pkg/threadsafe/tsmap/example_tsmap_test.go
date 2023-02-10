package tsmap_test

import (
	"fmt"
	"sync"

	"github.com/vonage/gosrvlib/pkg/threadsafe/tsmap"
)

func ExampleSet() {
	mux := &sync.Mutex{}

	m := make(map[int]string, 2)
	tsmap.Set(mux, m, 0, "Hello")
	tsmap.Set(mux, m, 1, "World")

	fmt.Println(m)

	// Output:
	// map[0:Hello 1:World]
}

func ExampleGet() {
	mux := &sync.RWMutex{}

	m := map[int]string{0: "Hello", 1: "World"}
	fmt.Println(tsmap.Get(mux, m, 0))
	fmt.Println(tsmap.Get(mux, m, 1))

	// Output:
	// Hello
	// World
}

func ExampleLen() {
	mux := &sync.RWMutex{}

	m := map[int]string{0: "Hello", 1: "World"}
	fmt.Println(tsmap.Len(mux, m))

	// Output:
	// 2
}
