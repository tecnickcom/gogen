package tsmap

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tecnickcom/nurago/pkg/maputil"
)

func TestGuarded_zeroValueSet(t *testing.T) {
	t.Parallel()

	// Zero value: backing map is nil and must be allocated lazily on write.
	var g Guarded[string, int]

	require.Equal(t, 0, g.Len())

	g.Set("a", 1)
	require.Equal(t, 1, g.Len())
	require.Equal(t, 1, g.Get("a"))
}

func TestGuarded_zeroValueDo(t *testing.T) {
	t.Parallel()

	var g Guarded[string, int]

	// Do must receive a non-nil map even from the zero value.
	g.Do(func(m map[string]int) {
		m["a"] = 1
	})

	require.Equal(t, 1, g.Get("a"))
}

func TestGuarded_SetGetDelete(t *testing.T) {
	t.Parallel()

	g := NewGuarded(map[string]int{"a": 1})

	g.Set("b", 2) // non-nil backing map path
	require.Equal(t, 2, g.Len())

	v, ok := g.GetOK("b")
	require.True(t, ok)
	require.Equal(t, 2, v)

	g.Delete("a")
	_, ok = g.GetOK("a")
	require.False(t, ok)

	g.Delete("missing") // no-op
	require.Equal(t, 1, g.Len())
}

func TestGuarded_GetMissing(t *testing.T) {
	t.Parallel()

	g := NewGuarded[string, int](nil)

	require.Equal(t, 0, g.Get("nope"))
}

func TestGuarded_Filter(t *testing.T) {
	t.Parallel()

	g := NewGuarded(map[int]string{0: "Hello", 1: "World"})

	got := g.Filter(func(_ int, v string) bool { return v == "World" })

	require.Len(t, got, 1)
	require.Equal(t, "World", got[1])
}

func TestGuarded_Snapshot(t *testing.T) {
	t.Parallel()

	g := NewGuarded(map[string]int{"a": 1, "b": 2})

	snap := g.Snapshot()
	snap["a"] = 99 // mutating the copy must not affect the guarded map

	require.Equal(t, 1, g.Get("a"))
}

func TestGuarded_Do(t *testing.T) {
	t.Parallel()

	g := NewGuarded(map[string]int{"a": 1})

	// Atomic check-then-set.
	g.Do(func(m map[string]int) {
		if _, ok := m["b"]; !ok {
			m["b"] = 2
		}

		m["a"]++
	})

	require.Equal(t, 2, g.Get("a"))
	require.Equal(t, 2, g.Get("b"))
}

func TestGuarded_RDo(t *testing.T) {
	t.Parallel()

	g := NewGuarded(map[int]int{1: 10, 2: 20})

	// Invert to a different type via RDo.
	var inv map[int]int

	g.RDo(func(m map[int]int) {
		inv = maputil.Invert(m)
	})

	require.Equal(t, 1, inv[10])
	require.Equal(t, 2, inv[20])
}

func TestGuarded_concurrent(t *testing.T) {
	t.Parallel()

	const iters = 1000

	g := NewGuarded[int, int](nil)

	wg := &sync.WaitGroup{}

	wg.Go(func() {
		for i := range iters {
			g.Set(i, i)
		}
	})

	for range 4 {
		wg.Go(func() {
			for range iters {
				_ = g.Len()
				_, _ = g.GetOK(0)
				_ = g.Filter(func(_ int, v int) bool { return v%2 == 0 })
			}
		})
	}

	wg.Wait()

	require.Equal(t, iters, g.Len())
}
