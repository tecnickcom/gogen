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
	Set(mux, &s, 0, "Hello")
	Set(mux, &s, 1, "World")

	require.ElementsMatch(t, []string{"Hello", "World"}, s)
}

func TestSetOK(t *testing.T) {
	t.Parallel()

	mux := &sync.Mutex{}

	s := make([]string, 2)

	require.True(t, SetOK(mux, &s, 0, "Hello"))
	require.False(t, SetOK(mux, &s, 2, "World"))
	require.False(t, SetOK(mux, &s, -1, "Nope"))
	require.Equal(t, "Hello", s[0])
	require.Empty(t, s[1])
}

func TestGet(t *testing.T) {
	t.Parallel()

	mux := &sync.RWMutex{}

	s := []string{"Hello", "World"}

	require.Equal(t, "Hello", Get(mux, &s, 0))
	require.Equal(t, "World", Get(mux, &s, 1))
}

func TestGetOK(t *testing.T) {
	t.Parallel()

	mux := &sync.RWMutex{}

	s := []string{"Hello", "World"}

	v, ok := GetOK(mux, &s, 1)
	require.True(t, ok)
	require.Equal(t, "World", v)

	v, ok = GetOK(mux, &s, 2)
	require.False(t, ok)
	require.Empty(t, v)

	v, ok = GetOK(mux, &s, -1)
	require.False(t, ok)
	require.Empty(t, v)
}

func TestLen(t *testing.T) {
	t.Parallel()

	mux := &sync.RWMutex{}

	s := []string{"Hello", "World"}

	require.Equal(t, 2, Len(mux, &s))
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

	maxgor := 5
	s := make([]int, 0, maxgor)

	for i := range maxgor {
		wg.Add(1)

		go func(item int) {
			defer wg.Done()

			Append(mux, &s, item)
		}(i)
	}

	wg.Wait()

	require.ElementsMatch(t, []int{0, 1, 2, 3, 4}, s)
}

// TestConcurrentAppendAndReaders exercises the read helpers concurrently with a
// goroutine that grows (and reallocates) the shared slice via Append. Because
// all helpers take *S and dereference under the lock, this is race-free; it is a
// regression guard for the historical by-value header race and must stay clean
// under `go test -race`.
func TestConcurrentAppendAndReaders(t *testing.T) {
	t.Parallel()

	const iters = 1000

	mux := &sync.RWMutex{}
	s := make([]int, 0, 1)

	wg := &sync.WaitGroup{}

	wg.Go(func() {
		for i := range iters {
			Append(mux, &s, i)
		}
	})

	for range 4 {
		wg.Go(func() {
			for range iters {
				_ = Len(mux, &s)
				_, _ = GetOK(mux, &s, 0)
				_ = Filter(mux, &s, func(_ int, v int) bool { return v%2 == 0 })
			}
		})
	}

	wg.Wait()

	require.Equal(t, iters, Len(mux, &s))
}

func TestFilter(t *testing.T) {
	t.Parallel()

	mux := &sync.RWMutex{}

	s := []string{"Hello", "World", "Extra"}
	filterFn := func(_ int, v string) bool { return v == "World" }

	got := Filter(mux, &s, filterFn)

	require.ElementsMatch(t, []string{"World"}, got)
}

func TestMap(t *testing.T) {
	t.Parallel()

	mux := &sync.RWMutex{}

	s := []string{"Hello", "World", "Extra"}
	mapFn := func(k int, v string) int { return k + len(v) }

	got := Map(mux, &s, mapFn)

	require.ElementsMatch(t, []int{5, 6, 7}, got)
}

func TestReduce(t *testing.T) {
	t.Parallel()

	mux := &sync.RWMutex{}

	s := []int{2, 3, 5, 7, 11}
	init := 97
	reduceFn := func(k, v, r int) int { return k + v + r }

	got := Reduce(mux, &s, init, reduceFn)

	require.Equal(t, 135, got)
}

func TestDo(t *testing.T) {
	t.Parallel()

	mux := &sync.Mutex{}

	s := []int{1, 2, 3}

	// Atomic append-if-shorter-than-target compound operation.
	Do(mux, &s, func(sp *[]int) {
		if len(*sp) < 4 {
			*sp = append(*sp, 4)
		}
	})

	require.Equal(t, []int{1, 2, 3, 4}, s)
}

func TestRDo(t *testing.T) {
	t.Parallel()

	mux := &sync.RWMutex{}

	s := []int{1, 2, 3, 4}

	sum := 0

	RDo(mux, &s, func(sv []int) {
		for _, v := range sv {
			sum += v
		}
	})

	require.Equal(t, 10, sum)
}

func TestDelete(t *testing.T) {
	t.Parallel()

	mux := &sync.Mutex{}

	s := []string{"a", "b", "c", "d"}

	require.True(t, Delete(mux, &s, 1))
	require.Equal(t, []string{"a", "c", "d"}, s)

	require.False(t, Delete(mux, &s, 5))
	require.False(t, Delete(mux, &s, -1))
	require.Equal(t, []string{"a", "c", "d"}, s)
}

func TestSnapshot(t *testing.T) {
	t.Parallel()

	mux := &sync.RWMutex{}

	s := []int{1, 2, 3}

	snap := Snapshot(mux, &s)
	snap[0] = 99 // mutating the copy must not affect the original

	require.Equal(t, []int{1, 2, 3}, s)
	require.Equal(t, []int{99, 2, 3}, snap)
}
