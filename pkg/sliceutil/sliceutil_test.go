package sliceutil

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFilter(t *testing.T) {
	t.Parallel()

	s := []string{"Hello", "World", "Extra"}
	filterFn := func(_ int, v string) bool { return v == "World" }

	got := Filter(s, filterFn)

	require.ElementsMatch(t, []string{"World"}, got)
}

func TestMap(t *testing.T) {
	t.Parallel()

	s := []string{"Hello", "World", "Extra"}
	mapFn := func(k int, v string) int { return k + len(v) }

	got := Map(s, mapFn)

	require.ElementsMatch(t, []int{5, 6, 7}, got)
}

func TestReduce(t *testing.T) {
	t.Parallel()

	s := []int{2, 3, 5, 7, 11}
	init := 97
	reduceFn := func(k, v, r int) int { return k + v + r }

	got := Reduce(s, init, reduceFn)

	require.Equal(t, 135, got)
}

func TestFilterNoMatch(t *testing.T) {
	t.Parallel()

	got := Filter([]int{1, 2, 3}, func(_, v int) bool { return v > 10 })

	require.NotNil(t, got)
	require.Empty(t, got)
}

func TestSliceNilInputs(t *testing.T) {
	t.Parallel()

	require.NotNil(t, Filter([]int(nil), func(_, _ int) bool { return true }))
	require.NotNil(t, Map([]int(nil), func(_, v int) int { return v }))
	require.Equal(t, 7, Reduce([]int(nil), 7, func(_, v, acc int) int { return v + acc }))
}
