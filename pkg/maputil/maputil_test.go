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
	require.Equal(t, "World", got[1])
	require.NotContains(t, got, 0)
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

func TestFilterNoMatch(t *testing.T) {
	t.Parallel()

	got := Filter(map[int]string{0: "Hello"}, func(_ int, _ string) bool { return false })

	require.NotNil(t, got)
	require.Empty(t, got)
}

func TestMapLastWriteWins(t *testing.T) {
	t.Parallel()

	m := map[int]string{1: "a", 2: "b"}
	// Both entries collide on the same output key: last write wins.
	got := Map(m, func(_ int, _ string) (string, int) { return "same", 1 })

	require.Len(t, got, 1)
	require.Equal(t, 1, got["same"])
}

func TestInvertLastWriteWins(t *testing.T) {
	t.Parallel()

	m := map[int]int{1: 9, 2: 9} // duplicate value 9 collides when inverted

	got := Invert(m)

	require.Len(t, got, 1)
	require.Contains(t, []int{1, 2}, got[9])
}

func TestMapUtilNilInputs(t *testing.T) {
	t.Parallel()

	require.NotNil(t, Filter(map[int]int(nil), func(_, _ int) bool { return true }))
	require.NotNil(t, Map(map[int]int(nil), func(k, v int) (int, int) { return k, v }))
	require.NotNil(t, Invert(map[int]int(nil)))
	require.Equal(t, 5, Reduce(map[int]int(nil), 5, func(_, v, acc int) int { return v + acc }))
}
