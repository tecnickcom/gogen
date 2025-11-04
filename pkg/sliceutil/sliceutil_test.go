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
