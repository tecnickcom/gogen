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

	c := New(resolver, int(1<<63-1), 1*time.Second)

	b.ResetTimer()

	for i := range b.N {
		_, _ = c.LookupHost(context.TODO(), strconv.Itoa(i)+testDomain)
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
		_, _ = c.LookupHost(context.TODO(), strconv.Itoa(i)+testDomain)
	}

	var j int

	b.ResetTimer()

	for range b.N {
		j++
		if j > size {
			j = 0
		}

		_, _ = c.LookupHost(context.TODO(), strconv.Itoa(j)+testDomain)
	}
}
