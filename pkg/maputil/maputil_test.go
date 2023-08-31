package maputil

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFilter(t *testing.T) {
	t.Parallel()

	m := map[int]string{0: "Hello", 1: "World"}
	filterFn := func(_ int, v string) bool { return v == "World" }

	got := Filter(m, filterFn)

	require.Len(t, got, 1)
	require.Equal(t, "World", m[1])
}

func TestMap(t *testing.T) {
	t.Parallel()

	m := map[int]string{0: "Hello", 1: "World"}
	mapFn := func(k int, v string) (string, int) { return "_" + v, k + 1 }

	got := Map(m, mapFn)

	require.Len(t, got, 2)
	require.Equal(t, 1, got["_Hello"])
	require.Equal(t, 2, got["_World"])
}

func TestReduce(t *testing.T) {
	t.Parallel()

	m := map[int]int{0: 2, 1: 3, 2: 5, 3: 7, 4: 11}
	init := 97
	reduceFn := func(k, v, r int) int { return k + v + r }

	got := Reduce(m, init, reduceFn)

	require.Equal(t, 135, got)
}

func TestInvert(t *testing.T) {
	t.Parallel()

	m := map[int]int{1: 10, 2: 20}

	got := Invert(m)

	require.Len(t, got, 2)
	require.Equal(t, 1, got[10])
	require.Equal(t, 2, got[20])
}
