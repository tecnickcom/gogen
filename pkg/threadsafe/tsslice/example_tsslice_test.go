package tsslice_test

import (
	"fmt"
	"sort"
	"sync"

	"github.com/Vonage/gosrvlib/pkg/threadsafe/tsslice"
)

func ExampleSet() {
	mux := &sync.Mutex{}

	slice := make([]string, 2)
	tsslice.Set(mux, slice, 0, "Hello")
	tsslice.Set(mux, slice, 1, "World")

	fmt.Println(slice)

	// Output:
	// [Hello World]
}

func ExampleGet() {
	mux := &sync.RWMutex{}

	slice := []string{"Hello", "World"}
	fmt.Println(tsslice.Get(mux, slice, 0))
	fmt.Println(tsslice.Get(mux, slice, 1))

	// Output:
	// Hello
	// World
}

func ExampleLen() {
	mux := &sync.RWMutex{}

	slice := []string{"Hello", "World"}
	fmt.Println(tsslice.Len(mux, slice))

	// Output:
	// 2
}

func ExampleAppend_simple() {
	mux := &sync.Mutex{}

	slice := make([]string, 0, 2)
	tsslice.Append(mux, &slice, "Hello")
	tsslice.Append(mux, &slice, "World")

	fmt.Println(slice)

	// Output:
	// [Hello World]
}

func ExampleAppend_multiple() {
	mux := &sync.Mutex{}

	slice := make([]string, 0, 2)
	tsslice.Append(mux, &slice, "Hello", "World")

	fmt.Println(slice)

	// Output:
	// [Hello World]
}

func ExampleAppend_slice() {
	mux := &sync.Mutex{}

	slice := make([]string, 0, 2)
	tsslice.Append(mux, &slice, []string{"Hello", "World"}...)

	fmt.Println(slice)

	// Output:
	// [Hello World]
}

func ExampleAppend_concurrent() {
	wg := &sync.WaitGroup{}
	mux := &sync.RWMutex{}

	max := 5
	slice := make([]int, 0, max)

	for i := 0; i < max; i++ {
		wg.Add(1)

		go func(item int) {
			defer wg.Done()

			tsslice.Append(mux, &slice, item)
		}(i)
	}

	wg.Wait()

	sort.Ints(slice)
	fmt.Println(slice)

	// Output:
	// [0 1 2 3 4]
}
