package dnscache

import (
	"context"
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

	r := New(resolver, int(1<<63-1), 1*time.Second)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = r.LookupHost(context.TODO(), strconv.Itoa(i)+testDomain)
	}
}

func BenchmarkLookupHost_cache_hit(b *testing.B) {
	resolver := &mockResolver{
		lookupHost: func(_ context.Context, _ string) ([]string, error) {
			return []string{"192.0.2.1"}, nil
		},
	}

	size := 255

	r := New(resolver, size, 1*time.Minute)

	// fill the cache
	for i := 1; i <= size; i++ {
		_, _ = r.LookupHost(context.TODO(), strconv.Itoa(i)+testDomain)
	}

	var j int

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		j++
		if j > size {
			j = 0
		}

		_, _ = r.LookupHost(context.TODO(), strconv.Itoa(j)+testDomain)
	}
}
