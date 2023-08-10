package tsslice_test

import (
	"fmt"
	"sort"
	"sync"

	"github.com/Vonage/gosrvlib/pkg/threadsafe/tsslice"
)

func ExampleSet() {
	mux := &sync.Mutex{}

	s := make([]string, 2)
	tsslice.Set(mux, s, 0, "Hello")
	tsslice.Set(mux, s, 1, "World")

	fmt.Println(s)

	// Output:
	// [Hello World]
}

func ExampleGet() {
	mux := &sync.RWMutex{}

	s := []string{"Hello", "World"}
	fmt.Println(tsslice.Get(mux, s, 0))
	fmt.Println(tsslice.Get(mux, s, 1))

	// Output:
	// Hello
	// World
}

func ExampleLen() {
	mux := &sync.RWMutex{}

	s := []string{"Hello", "World"}
	fmt.Println(tsslice.Len(mux, s))

	// Output:
	// 2
}

func ExampleAppend_simple() {
	mux := &sync.Mutex{}

	s := make([]string, 0, 2)
	tsslice.Append(mux, &s, "Hello")
	tsslice.Append(mux, &s, "World")

	fmt.Println(s)

	// Output:
	// [Hello World]
}

func ExampleAppend_multiple() {
	mux := &sync.Mutex{}

	s := make([]string, 0, 2)
	tsslice.Append(mux, &s, "Hello", "World")

	fmt.Println(s)

	// Output:
	// [Hello World]
}

func ExampleAppend_slice() {
	mux := &sync.Mutex{}

	s := make([]string, 0, 2)
	tsslice.Append(mux, &s, []string{"Hello", "World"}...)

	fmt.Println(s)

	// Output:
	// [Hello World]
}

func ExampleAppend_concurrent() {
	wg := &sync.WaitGroup{}
	mux := &sync.RWMutex{}

	max := 5
	s := make([]int, 0, max)

	for i := 0; i < max; i++ {
		wg.Add(1)

		go func(item int) {
			defer wg.Done()

			tsslice.Append(mux, &s, item)
		}(i)
	}

	wg.Wait()

	sort.Ints(s)
	fmt.Println(s)

	// Output:
	// [0 1 2 3 4]
}

func ExampleFilter() {
	mux := &sync.RWMutex{}

	s := []string{"Hello", "World", "Extra"}

	filterFn := func(_ int, v string) bool { return v == "World" }

	s2 := tsslice.Filter(mux, s, filterFn)

	fmt.Println(s2)

	// Output:
	// [World]
}

func ExampleMap() {
	mux := &sync.RWMutex{}

	s := []string{"Hello", "World", "Extra"}

	mapFn := func(k int, v string) int { return k + len(v) }

	s2 := tsslice.Map(mux, s, mapFn)

	fmt.Println(s2)

	// Output:
	// [5 6 7]
}

func ExampleReduce() {
	mux := &sync.RWMutex{}

	s := []int{2, 3, 5, 7, 11}

	init := 97
	reduceFn := func(k, v, r int) int { return k + v + r }

	r := tsslice.Reduce(mux, s, init, reduceFn)

	fmt.Println(r)

	// Output:
	// 135
}
