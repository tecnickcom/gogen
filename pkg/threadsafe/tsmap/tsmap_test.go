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

func TestGet(t *testing.T) {
	t.Parallel()

	mux := &sync.RWMutex{}

	m := map[int]string{0: "Hello", 1: "World"}

	require.Equal(t, "Hello", Get(mux, m, 0))
	require.Equal(t, "World", Get(mux, m, 1))
}

func TestLen(t *testing.T) {
	t.Parallel()

	mux := &sync.RWMutex{}

	m := map[int]string{0: "Hello", 1: "World"}

	require.Equal(t, 2, Len(mux, m))
}
