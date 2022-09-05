package threadsafe_test

import (
	"fmt"
	"sort"
	"sync"

	"github.com/nexmoinc/gosrvlib/pkg/threadsafe"
)

func ExampleAppend_simple() {
	mux := &sync.Mutex{}

	slice := make([]string, 0, 2)
	threadsafe.Append(mux, &slice, "Hello")
	threadsafe.Append(mux, &slice, "World")

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

			threadsafe.Append(mux, &slice, item)
		}(i)
	}

	wg.Wait()

	sort.Ints(slice)
	fmt.Println(slice)

	// Output:
	// [0 1 2 3 4]
}
