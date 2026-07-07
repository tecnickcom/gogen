package dnscache

import (
	"context"
	"math"
	"strconv"
	"testing"
	"time"
)

const testDomain = "example.com"

func BenchmarkLookupHost_cache_miss(b *testing.B) {
	resolver := &mockResolver{
		lookupHost: func(_ context.Context, _ string) ([]string, error) {
			return []string{"192.0.2.1"}, nil
		},
	}

	c := New(resolver, math.MaxInt, 1*time.Second)

	b.ResetTimer()

	for i := range b.N {
		_, _ = c.LookupHost(b.Context(), strconv.Itoa(i)+testDomain)
	}
}

func BenchmarkLookupHost_cache_hit(b *testing.B) {
	resolver := &mockResolver{
		lookupHost: func(_ context.Context, _ string) ([]string, error) {
			return []string{"192.0.2.1"}, nil
		},
	}

	size := 255

	c := New(resolver, size, 1*time.Minute)

	// fill the cache
	for i := 1; i <= size; i++ {
		_, _ = c.LookupHost(b.Context(), strconv.Itoa(i)+testDomain)
	}

	var j int

	for b.Loop() {
		j++
		if j > size {
			j = 0
		}

		_, _ = c.LookupHost(b.Context(), strconv.Itoa(j)+testDomain)
	}
}

func BenchmarkNormalizeHost_lower(b *testing.B) {
	for b.Loop() {
		_ = normalizeHost("cache.example.com")
	}
}

func BenchmarkNormalizeHost_mixed(b *testing.B) {
	for b.Loop() {
		_ = normalizeHost("Cache.Example.COM.")
	}
}

func BenchmarkOrderCandidates_single_family(b *testing.B) {
	c := New(nil, 1, 1*time.Minute)
	ips := []string{"192.0.2.1", "192.0.2.2", "192.0.2.3", "192.0.2.4"}

	for b.Loop() {
		_ = c.orderCandidates("tcp", ips)
	}
}

func BenchmarkOrderCandidates_mixed_family(b *testing.B) {
	c := New(nil, 1, 1*time.Minute)
	ips := []string{"2001:db8::1", "192.0.2.1", "2001:db8::2", "192.0.2.2"}

	for b.Loop() {
		_ = c.orderCandidates("tcp", ips)
	}
}
