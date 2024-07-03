package tsslice

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSet(t *testing.T) {
	t.Parallel()

	mux := &sync.Mutex{}

	s := make([]string, 2)
	Set(mux, s, 0, "Hello")
	Set(mux, s, 1, "World")

	require.ElementsMatch(t, []string{"Hello", "World"}, s)
}

func TestGet(t *testing.T) {
	t.Parallel()

	mux := &sync.RWMutex{}

	s := []string{"Hello", "World"}

	require.Equal(t, "Hello", Get(mux, s, 0))
	require.Equal(t, "World", Get(mux, s, 1))
}

func TestLen(t *testing.T) {
	t.Parallel()

	mux := &sync.RWMutex{}

	s := []string{"Hello", "World"}

	require.Equal(t, 2, Len(mux, s))
}

func TestAppend_simple(t *testing.T) {
	t.Parallel()

	mux := &sync.Mutex{}

	s := make([]string, 0, 2)
	Append(mux, &s, "Hello")
	Append(mux, &s, "World")

	require.ElementsMatch(t, []string{"Hello", "World"}, s)
}

func TestAppend_multiple(t *testing.T) {
	t.Parallel()

	mux := &sync.Mutex{}

	s := make([]string, 0, 2)
	Append(mux, &s, "Hello", "World")

	require.ElementsMatch(t, []string{"Hello", "World"}, s)
}

func TestAppend_slice(t *testing.T) {
	t.Parallel()

	mux := &sync.Mutex{}

	s := make([]string, 0, 2)
	Append(mux, &s, []string{"Hello", "World"}...)

	require.ElementsMatch(t, []string{"Hello", "World"}, s)
}

func TestAppend_concurrent(t *testing.T) {
	t.Parallel()

	wg := &sync.WaitGroup{}
	mux := &sync.RWMutex{}

	max := 5
	s := make([]int, 0, max)

	for i := range max {
		wg.Add(1)

		go func(item int) {
			defer wg.Done()

			Append(mux, &s, item)
		}(i)
	}

	wg.Wait()

	require.ElementsMatch(t, []int{0, 1, 2, 3, 4}, s)
}

func TestFilter(t *testing.T) {
	t.Parallel()

	mux := &sync.RWMutex{}

	s := []string{"Hello", "World", "Extra"}
	filterFn := func(_ int, v string) bool { return v == "World" }

	got := Filter(mux, s, filterFn)

	require.ElementsMatch(t, []string{"World"}, got)
}

func TestMap(t *testing.T) {
	t.Parallel()

	mux := &sync.RWMutex{}

	s := []string{"Hello", "World", "Extra"}
	mapFn := func(k int, v string) int { return k + len(v) }

	got := Map(mux, s, mapFn)

	require.ElementsMatch(t, []int{5, 6, 7}, got)
}

func TestReduce(t *testing.T) {
	t.Parallel()

	mux := &sync.RWMutex{}

	s := []int{2, 3, 5, 7, 11}
	init := 97
	reduceFn := func(k, v, r int) int { return k + v + r }

	got := Reduce(mux, s, init, reduceFn)

	require.Equal(t, 135, got)
}
