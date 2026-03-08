package random

import (
	"sync"
	"testing"
)

func TestNewID64(t *testing.T) {
	t.Parallel()

	r := New(nil)

	a := r.UID64()
	b := r.UID64()

	if a == b {
		t.Errorf("Two UID should be different")
	}
}

func TestNewID64_Collision(t *testing.T) {
	t.Parallel()

	r := New(nil)

	collisionTest(t, r.UID64, 10, 100)
}

func TestNewID128(t *testing.T) {
	t.Parallel()

	r := New(nil)

	a := r.UID64()
	b := r.UID64()

	if a == b {
		t.Errorf("Two UID should be different")
	}
}

func TestNewID128_Collision(t *testing.T) {
	t.Parallel()

	r := New(nil)

	collisionTest(t, r.UID128, 100, 1000)
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
