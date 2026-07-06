package tsslice

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tecnickcom/gogen/pkg/sliceutil"
)

func TestGuarded_zeroValue(t *testing.T) {
	t.Parallel()

	var g Guarded[int]

	require.Equal(t, 0, g.Len())

	g.Append(1, 2)
	require.Equal(t, 2, g.Len())
}

func TestGuarded_LenGetSet(t *testing.T) {
	t.Parallel()

	g := NewGuarded([]string{"Hello", "World"})

	require.Equal(t, 2, g.Len())
	require.Equal(t, "Hello", g.Get(0))

	g.Set(1, "Gophers")
	require.Equal(t, "Gophers", g.Get(1))
}

func TestGuarded_GetOK(t *testing.T) {
	t.Parallel()

	g := NewGuarded([]string{"Hello", "World"})

	v, ok := g.GetOK(1)
	require.True(t, ok)
	require.Equal(t, "World", v)

	v, ok = g.GetOK(2)
	require.False(t, ok)
	require.Empty(t, v)

	v, ok = g.GetOK(-1)
	require.False(t, ok)
	require.Empty(t, v)
}

func TestGuarded_SetOK(t *testing.T) {
	t.Parallel()

	g := NewGuarded(make([]string, 2))

	require.True(t, g.SetOK(0, "Hello"))
	require.False(t, g.SetOK(2, "World"))
	require.False(t, g.SetOK(-1, "Nope"))
	require.Equal(t, "Hello", g.Get(0))
}

func TestGuarded_Append(t *testing.T) {
	t.Parallel()

	g := NewGuarded[int](nil)

	g.Append(1)
	g.Append(2, 3)

	require.Equal(t, []int{1, 2, 3}, g.Snapshot())
}

func TestGuarded_Delete(t *testing.T) {
	t.Parallel()

	g := NewGuarded([]string{"a", "b", "c", "d"})

	require.True(t, g.Delete(1))
	require.Equal(t, []string{"a", "c", "d"}, g.Snapshot())

	require.False(t, g.Delete(9))
	require.False(t, g.Delete(-1))
	require.Equal(t, []string{"a", "c", "d"}, g.Snapshot())
}

func TestGuarded_Filter(t *testing.T) {
	t.Parallel()

	g := NewGuarded([]string{"Hello", "World", "Extra"})

	got := g.Filter(func(_ int, v string) bool { return v == "World" })

	require.Equal(t, []string{"World"}, got)
}

func TestGuarded_Snapshot(t *testing.T) {
	t.Parallel()

	g := NewGuarded([]int{1, 2, 3})

	snap := g.Snapshot()
	snap[0] = 99 // mutating the copy must not affect the guarded slice

	require.Equal(t, []int{1, 2, 3}, g.Snapshot())
}

func TestGuarded_Do(t *testing.T) {
	t.Parallel()

	g := NewGuarded([]int{1, 2, 3})

	g.Do(func(s *[]int) {
		if len(*s) < 4 {
			*s = append(*s, 4)
		}
	})

	require.Equal(t, []int{1, 2, 3, 4}, g.Snapshot())
}

func TestGuarded_RDo(t *testing.T) {
	t.Parallel()

	g := NewGuarded([]int{1, 2, 3, 4})

	// Reduce to a different type via RDo.
	sum := 0

	g.RDo(func(s []int) {
		sum = sliceutil.Reduce(s, 0, func(_ int, v, acc int) int { return acc + v })
	})

	require.Equal(t, 10, sum)
}

func TestGuarded_concurrent(t *testing.T) {
	t.Parallel()

	const iters = 1000

	g := NewGuarded(make([]int, 0, 1))

	wg := &sync.WaitGroup{}

	wg.Go(func() {
		for i := range iters {
			g.Append(i)
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
