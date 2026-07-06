package tsmap_test

import (
	"fmt"

	"github.com/tecnickcom/gogen/pkg/maputil"
	"github.com/tecnickcom/gogen/pkg/threadsafe/tsmap"
)

func ExampleGuarded() {
	// The guard owns both the map and its lock; every method is safe to call
	// from multiple goroutines without pairing a separate mutex.
	g := tsmap.NewGuarded(map[string]int{"a": 1})

	g.Set("b", 2)
	g.Delete("a")

	fmt.Println(g.Len())

	v, ok := g.GetOK("b")
	fmt.Println(v, ok)

	// Output:
	// 1
	// 2 true
}

func ExampleGuarded_Do() {
	// Do runs a compound operation atomically under the write lock.
	g := tsmap.NewGuarded(map[string]int{"a": 1})

	g.Do(func(m map[string]int) {
		if _, ok := m["b"]; !ok {
			m["b"] = 2
		}

		m["a"]++
	})

	fmt.Println(g.Get("a"), g.Get("b"))

	// Output:
	// 2 2
}

func ExampleGuarded_RDo() {
	// Transforms such as Invert go through RDo, which holds the read lock while
	// delegating to maputil.
	g := tsmap.NewGuarded(map[int]int{1: 10, 2: 20})

	var inv map[int]int

	g.RDo(func(m map[int]int) {
		inv = maputil.Invert(m)
	})

	fmt.Println(inv[10], inv[20])

	// Output:
	// 1 2
}
