package sfcache

import (
	"context"
	"math"
	"strconv"
	"testing"
	"time"
)

const testDomain = "example.com"

// benchKeys pre-generates n distinct keys so that key construction never
// pollutes the timed benchmark loops.
func benchKeys(n int) []string {
	keys := make([]string, n)

	for i := range n {
		keys[i] = strconv.Itoa(i) + testDomain
	}

	return keys
}

// fillCache warms the cache with all keys, failing the benchmark on error.
func fillCache(b *testing.B, c *Cache[string, []string], keys []string) {
	b.Helper()

	ctx := context.Background()

	for _, k := range keys {
		_, err := c.Lookup(ctx, k)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func benchLookupFn(_ context.Context, _ string) ([]string, error) {
	return []string{"192.0.2.1"}, nil
}

// BenchmarkLookup_cache_miss measures the pure miss path on an unbounded
// cache: every key is new and no eviction ever happens.
func BenchmarkLookup_cache_miss(b *testing.B) {
	c := New(benchLookupFn, math.MaxInt, 1*time.Hour)
	keys := benchKeys(b.N)
	ctx := context.Background()

	b.ResetTimer()

	for i := range b.N {
		_, err := c.Lookup(ctx, keys[i])
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkLookup_cache_miss_at_capacity measures the steady-state miss path
// on a full cache: every miss pays the O(size) eviction scan.
func BenchmarkLookup_cache_miss_at_capacity(b *testing.B) {
	size := 255

	c := New(benchLookupFn, size, 1*time.Hour)
	ctx := context.Background()

	all := benchKeys(b.N + size)
	fillCache(b, c, all[:size])

	keys := all[size:]

	b.ResetTimer()

	for i := range b.N {
		_, err := c.Lookup(ctx, keys[i])
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkLookup_cache_hit measures single-goroutine hits cycling over the
// full key set of a warm cache.
func BenchmarkLookup_cache_hit(b *testing.B) {
	size := 255

	c := New(benchLookupFn, size, 1*time.Hour)
	ctx := context.Background()
	keys := benchKeys(size)

	fillCache(b, c, keys)

	var j int

	for b.Loop() {
		j++
		if j >= size {
			j = 0
		}

		_, err := c.Lookup(ctx, keys[j])
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkLookup_cache_hit_parallel measures contended hits on a single hot
// key across all CPUs.
func BenchmarkLookup_cache_hit_parallel(b *testing.B) {
	c := New(benchLookupFn, 16, 1*time.Hour)
	ctx := context.Background()

	fillCache(b, c, []string{testDomain})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := c.Lookup(ctx, testDomain)
			if err != nil {
				b.Error(err)

				return
			}
		}
	})
}

// BenchmarkLookup_cache_hit_parallel_keys measures parallel hits spread over
// the full key set of a warm cache (read-lock scaling across keys).
func BenchmarkLookup_cache_hit_parallel_keys(b *testing.B) {
	size := 255

	c := New(benchLookupFn, size, 1*time.Hour)
	ctx := context.Background()
	keys := benchKeys(size)

	fillCache(b, c, keys)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		var j int

		for pb.Next() {
			j++
			if j >= size {
				j = 0
			}

			_, err := c.Lookup(ctx, keys[j])
			if err != nil {
				b.Error(err)

				return
			}
		}
	})
}

// BenchmarkLookup_refresh measures the refresh path: with a zero TTL every
// lookup for the same key runs the full placeholder + lookup + publish cycle.
func BenchmarkLookup_refresh(b *testing.B) {
	c := New(benchLookupFn, 16, 0)
	ctx := context.Background()

	fillCache(b, c, []string{testDomain})

	b.ResetTimer()

	for range b.N {
		_, err := c.Lookup(ctx, testDomain)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkLookup_refresh_stale measures the refresh path with
// WithStaleIfError enabled: each refresh of a previously good key pays the
// stale carry (staleFrom plus one extra placeholder allocation).
func BenchmarkLookup_refresh_stale(b *testing.B) {
	c := New(benchLookupFn, 16, 0, WithStaleIfError[string, []string](1*time.Hour))
	ctx := context.Background()

	fillCache(b, c, []string{testDomain})

	b.ResetTimer()

	for range b.N {
		_, err := c.Lookup(ctx, testDomain)
		if err != nil {
			b.Fatal(err)
		}
	}
}
