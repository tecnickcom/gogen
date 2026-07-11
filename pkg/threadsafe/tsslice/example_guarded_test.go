package tsslice_test

import (
	"fmt"

	"github.com/tecnickcom/nurago/pkg/sliceutil"
	"github.com/tecnickcom/nurago/pkg/threadsafe/tsslice"
)

func ExampleGuarded() {
	// The guard owns both the slice and its lock; every method is safe to call
	// from multiple goroutines without pairing a separate mutex.
	g := tsslice.NewGuarded([]int{1, 2, 3})

	g.Append(4, 5)
	g.Set(0, 10)
	g.Delete(1) // removes value 2, preserving order

	fmt.Println(g.Snapshot())
	fmt.Println(g.Len())

	v, ok := g.GetOK(0)
	fmt.Println(v, ok)

	// Output:
	// [10 3 4 5]
	// 4
	// 10 true
}

func ExampleGuarded_RDo() {
	// Transforms to a different element type go through RDo, which holds the
	// read lock while delegating to sliceutil.
	g := tsslice.NewGuarded([]int{1, 2, 3, 4})

	var lengths []int

	g.RDo(func(s []int) {
		lengths = sliceutil.Map(s, func(_ int, v int) int { return v * v })
	})

	fmt.Println(lengths)

	// Output:
	// [1 4 9 16]
}
