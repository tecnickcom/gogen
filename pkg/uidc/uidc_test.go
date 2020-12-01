package uidc

import (
	"sync"
	"testing"
)

func TestNewID64(t *testing.T) {
	a := NewID64()
	b := NewID64()
	if a == b {
		t.Errorf("Two UID should be different")
	}
}

func BenchmarkNewID64(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewID64()
	}
}

func TestNewID64_Collision(t *testing.T) {
	collisionTest(t, NewID64, 10, 1000)
}

func TestNewID128(t *testing.T) {
	a := NewID128()
	b := NewID128()
	if a == b {
		t.Errorf("Two UID should be different")
	}
}

func BenchmarkNewID128(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewID128()
	}
}

func TestNewID128_Collision(t *testing.T) {
	collisionTest(t, NewID128, 100, 10_000)
}

func collisionTest(t *testing.T, f func() string, concurrency, iterations int) {
	t.Parallel()

	total := concurrency * iterations

	idCh := make(chan string, total)
	defer close(idCh)

	// generators
	genWg := &sync.WaitGroup{}
	genWg.Add(concurrency)

	for i := 0; i < concurrency; i++ {
		go func() {
			defer genWg.Done()
			for i := 0; i < iterations; i++ {
				idCh <- f()
			}
		}()
	}

	// wait for generators to finish
	genWg.Wait()

	ids := make(map[string]bool, total)
	for i := 0; i < total; i++ {
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
