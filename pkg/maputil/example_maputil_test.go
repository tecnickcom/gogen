package maputil_test

import (
	"fmt"

	"github.com/Vonage/gosrvlib/pkg/maputil"
)

func ExampleFilter() {
	m := map[int]string{0: "Hello", 1: "World"}

	filterFn := func(_ int, v string) bool { return v == "World" }

	s2 := maputil.Filter(m, filterFn)

	fmt.Println(s2)

	// Output:
	// map[1:World]
}

func ExampleMap() {
	m := map[int]string{0: "Hello", 1: "World"}

	mapFn := func(k int, v string) (string, int) { return "_" + v, k + 1 }

	s2 := maputil.Map(m, mapFn)

	fmt.Println(s2)

	// Output:
	// map[_Hello:1 _World:2]
}

func ExampleReduce() {
	m := map[int]int{0: 2, 1: 3, 2: 5, 3: 7, 4: 11}
	init := 97
	reduceFn := func(k, v, r int) int { return k + v + r }

	r := maputil.Reduce(m, init, reduceFn)

	fmt.Println(r)

	// Output:
	// 135
}

func ExampleInvert() {
	m := map[int]int{1: 10, 2: 20}

	s2 := maputil.Invert(m)

	fmt.Println(s2)

	// Output:
	// map[10:1 20:2]
}
