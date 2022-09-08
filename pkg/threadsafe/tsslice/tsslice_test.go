package tsslice

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAppend_simple(t *testing.T) {
	t.Parallel()

	mux := &sync.Mutex{}

	slice := make([]string, 0, 2)
	Append(mux, &slice, "Hello")
	Append(mux, &slice, "World")

	require.ElementsMatch(t, []string{"Hello", "World"}, slice)
}

func TestAppend_multiple(t *testing.T) {
	t.Parallel()

	mux := &sync.Mutex{}

	slice := make([]string, 0, 2)
	Append(mux, &slice, "Hello", "World")

	require.ElementsMatch(t, []string{"Hello", "World"}, slice)
}

func TestAppend_slice(t *testing.T) {
	t.Parallel()

	mux := &sync.Mutex{}

	slice := make([]string, 0, 2)
	Append(mux, &slice, []string{"Hello", "World"}...)

	require.ElementsMatch(t, []string{"Hello", "World"}, slice)
}

func TestAppend_concurrent(t *testing.T) {
	t.Parallel()

	wg := &sync.WaitGroup{}
	mux := &sync.RWMutex{}

	max := 5
	slice := make([]int, 0, max)

	for i := 0; i < max; i++ {
		wg.Add(1)

		go func(item int) {
			defer wg.Done()

			Append(mux, &slice, item)
		}(i)
	}

	wg.Wait()

	require.ElementsMatch(t, []int{0, 1, 2, 3, 4}, slice)
}

func TestSet(t *testing.T) {
	t.Parallel()

	mux := &sync.Mutex{}

	slice := make([]string, 2)
	Set(mux, slice, 0, "Hello")
	Set(mux, slice, 1, "World")

	require.ElementsMatch(t, []string{"Hello", "World"}, slice)
}

func TestGet(t *testing.T) {
	t.Parallel()

	mux := &sync.RWMutex{}

	slice := []string{"Hello", "World"}

	require.Equal(t, "Hello", Get(mux, slice, 0))
	require.Equal(t, "World", Get(mux, slice, 1))
}

func TestLen(t *testing.T) {
	t.Parallel()

	mux := &sync.RWMutex{}

	slice := []string{"Hello", "World"}

	require.Equal(t, 2, Len(mux, slice))
}
