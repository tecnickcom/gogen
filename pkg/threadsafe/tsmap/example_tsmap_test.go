package tsmap_test

import (
	"fmt"
	"sync"

	"github.com/tecnickcom/nurago/pkg/threadsafe/tsmap"
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

func ExampleDelete() {
	mux := &sync.Mutex{}

	m := map[int]string{0: "Hello", 1: "World"}

	tsmap.Delete(mux, m, 0)

	fmt.Println(m)

	// Output:
	// map[1:World]
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

func ExampleGetOK() {
	mux := &sync.RWMutex{}

	m := map[int]string{0: "Hello", 1: "World"}

	v, ok := tsmap.GetOK(mux, m, 0)
	fmt.Println(v, ok)

	v, ok = tsmap.GetOK(mux, m, 3)
	fmt.Println(v, ok)

	// Output:
	// Hello true
	//  false
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

func ExampleInvert() {
	mux := &sync.RWMutex{}

	m := map[int]int{1: 10, 2: 20}

	s2 := tsmap.Invert(mux, m)

	fmt.Println(s2)

	// Output:
	// map[10:1 20:2]
}

func ExampleDo() {
	mux := &sync.Mutex{}

	m := map[string]int{"a": 1}

	// Atomically set "b" only if it is absent, then bump "a".
	tsmap.Do(mux, m, func(mm map[string]int) {
		if _, ok := mm["b"]; !ok {
			mm["b"] = 2
		}

		mm["a"]++
	})

	fmt.Println(m["a"], m["b"])

	// Output:
	// 2 2
}

func ExampleRDo() {
	mux := &sync.RWMutex{}

	m := map[string]int{"a": 1, "b": 2, "c": 3}

	sum := 0

	tsmap.RDo(mux, m, func(mm map[string]int) {
		for _, v := range mm {
			sum += v
		}
	})

	fmt.Println(sum)

	// Output:
	// 6
}

func ExampleSnapshot() {
	mux := &sync.RWMutex{}

	m := map[string]int{"a": 1, "b": 2}

	// Snapshot returns a copy that is safe to use after the lock is released.
	snap := tsmap.Snapshot(mux, m)

	fmt.Println(len(snap), snap["a"], snap["b"])

	// Output:
	// 2 1 2
}
