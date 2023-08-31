package sliceutil_test

import (
	"fmt"

	"github.com/Vonage/gosrvlib/pkg/sliceutil"
)

func ExampleFilter() {
	s := []string{"Hello", "World", "Extra"}

	filterFn := func(_ int, v string) bool { return v == "World" }

	s2 := sliceutil.Filter(s, filterFn)

	fmt.Println(s2)

	// Output:
	// [World]
}

func ExampleMap() {
	s := []string{"Hello", "World", "Extra"}

	mapFn := func(k int, v string) int { return k + len(v) }

	s2 := sliceutil.Map(s, mapFn)

	fmt.Println(s2)

	// Output:
	// [5 6 7]
}

func ExampleReduce() {
	s := []int{2, 3, 5, 7, 11}

	init := 97
	reduceFn := func(k, v, r int) int { return k + v + r }

	r := sliceutil.Reduce(s, init, reduceFn)

	fmt.Println(r)

	// Output:
	// 135
}
