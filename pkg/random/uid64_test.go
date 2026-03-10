package random

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRnd_UID64(t *testing.T) {
	t.Parallel()

	r := New(nil)

	a := r.UID64()
	b := r.UID64()

	require.NotEqual(t, a, b)
}

func TestRnd_UID64_Hex(t *testing.T) {
	t.Parallel()

	r := New(nil)

	a := r.UID64().Hex()
	b := r.UID64().Hex()

	require.NotEqual(t, a, b)
	require.Len(t, a, 16)
	require.Len(t, b, 16)
}

func TestRnd_UID64_String(t *testing.T) {
	t.Parallel()

	r := New(nil)

	a := r.UID64().String()
	b := r.UID64().String()

	require.NotEqual(t, a, b)
}

func TestRnd_UID64_Hex_Collision(t *testing.T) {
	t.Parallel()

	r := New(nil)

	fn := func() string {
		return r.UID64().Hex()
	}

	collisionTest(t, fn, 10, 100)
}

func collisionTest(t *testing.T, f func() string, concurrency, iterations int) {
	t.Helper()

	total := concurrency * iterations

	idCh := make(chan string, total)
	defer close(idCh)

	// generators
	genWg := &sync.WaitGroup{}
	genWg.Add(concurrency)

	for range concurrency {
		go func() {
			defer genWg.Done()

			for range iterations {
				idCh <- f()
			}
		}()
	}

	// wait for generators to finish
	genWg.Wait()

	ids := make(map[string]bool, total)

	for range total {
		id, ok := <-idCh
		if !ok {
			t.Errorf("unexpected closed id channel")
			return
		}

		if _, exists := ids[id]; exists {
			t.Errorf("unexpected duplicate ID detected")
			return
		}

		// store generated id for duplicate detection
		ids[id] = true
	}
}
