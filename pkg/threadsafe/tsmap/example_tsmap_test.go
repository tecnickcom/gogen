package tsmap_test

import (
	"fmt"
	"sync"

	"github.com/Vonage/gosrvlib/pkg/threadsafe/tsmap"
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

func ExampleFilter() {
	mux := &sync.RWMutex{}

	m := map[int]string{0: "Hello", 1: "World"}

	filterFn := func(_ int, v string) bool { return v == "World" }

	s2 := tsmap.Filter(mux, m, filterFn)

	fmt.Println(s2)

	// Output:
	// map[1:World]
}

func ExampleMap() {
	mux := &sync.RWMutex{}

	m := map[int]string{0: "Hello", 1: "World"}

	mapFn := func(k int, v string) (string, int) { return "_" + v, k + 1 }

	s2 := tsmap.Map(mux, m, mapFn)

	fmt.Println(s2)

	// Output:
	// map[_Hello:1 _World:2]
}

func ExampleReduce() {
	mux := &sync.RWMutex{}

	m := map[int]int{0: 2, 1: 3, 2: 5, 3: 7, 4: 11}
	init := 97
	reduceFn := func(k, v, r int) int { return k + v + r }

	r := tsmap.Reduce(mux, m, init, reduceFn)

	fmt.Println(r)

	// Output:
	// 135
}
