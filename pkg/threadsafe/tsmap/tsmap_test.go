package tsmap

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSet(t *testing.T) {
	t.Parallel()

	mux := &sync.Mutex{}

	m := make(map[int]string, 2)
	Set(mux, m, 0, "Hello")
	Set(mux, m, 1, "World")

	require.Equal(t, "Hello", m[0])
	require.Equal(t, "World", m[1])
}

func TestDelete(t *testing.T) {
	t.Parallel()

	mux := &sync.Mutex{}

	m := map[int]string{3: "Hello", 5: "World"}

	v, ok := m[5]
	require.True(t, ok)
	require.Equal(t, "World", v)

	Delete(mux, m, 5)

	v, ok = m[5]
	require.False(t, ok)
	require.Equal(t, "", v)
}

func TestGet(t *testing.T) {
	t.Parallel()

	mux := &sync.RWMutex{}

	m := map[int]string{0: "Hello", 1: "World"}

	require.Equal(t, "Hello", Get(mux, m, 0))
	require.Equal(t, "World", Get(mux, m, 1))
	require.Equal(t, "", Get(mux, m, 3))
}

func TestGetOK(t *testing.T) {
	t.Parallel()

	mux := &sync.RWMutex{}

	m := map[int]string{5: "Hello", 7: "World"}

	v, ok := GetOK(mux, m, 7)
	require.True(t, ok)
	require.Equal(t, "World", v)

	v, ok = GetOK(mux, m, 6)
	require.False(t, ok)
	require.Equal(t, "", v)
}

func TestLen(t *testing.T) {
	t.Parallel()

	mux := &sync.RWMutex{}

	m := map[int]string{0: "Hello", 1: "World"}

	require.Equal(t, 2, Len(mux, m))
}

func TestFilter(t *testing.T) {
	t.Parallel()

	mux := &sync.RWMutex{}

	m := map[int]string{0: "Hello", 1: "World"}
	filterFn := func(_ int, v string) bool { return v == "World" }

	got := Filter(mux, m, filterFn)

	require.Len(t, got, 1)
	require.Equal(t, "World", m[1])
}

func TestMap(t *testing.T) {
	t.Parallel()

	mux := &sync.RWMutex{}

	m := map[int]string{0: "Hello", 1: "World"}
	mapFn := func(k int, v string) (string, int) { return "_" + v, k + 1 }

	got := Map(mux, m, mapFn)

	require.Len(t, got, 2)
	require.Equal(t, 1, got["_Hello"])
	require.Equal(t, 2, got["_World"])
}

func TestReduce(t *testing.T) {
	t.Parallel()

	mux := &sync.RWMutex{}

	m := map[int]int{0: 2, 1: 3, 2: 5, 3: 7, 4: 11}
	init := 97
	reduceFn := func(k, v, r int) int { return k + v + r }

	got := Reduce(mux, m, init, reduceFn)

	require.Equal(t, 135, got)
}

func TestInvert(t *testing.T) {
	t.Parallel()

	mux := &sync.RWMutex{}

	m := map[int]int{1: 10, 2: 20}

	got := Invert(mux, m)

	require.Len(t, got, 2)
	require.Equal(t, 1, got[10])
	require.Equal(t, 2, got[20])
}
