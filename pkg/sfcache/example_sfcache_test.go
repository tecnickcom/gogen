package sfcache_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/tecnickcom/gogen/pkg/sfcache"
)

func ExampleCache_Lookup() {
	// example lookup function that returns the key as value:
	// the cache value type V is inferred from its return type.
	lookupFn := func(_ context.Context, key string) (string, error) {
		return key, nil
	}

	// create a new cache with a lookupFn function, a maximum number of 3 entries, and a TTL of 1 minute.
	c := sfcache.New(lookupFn, 3, 1*time.Minute)

	val, err := c.Lookup(context.TODO(), "some_key")

	fmt.Println(val, err)

	// Output:
	// some_key <nil>
}

func ExampleWithTTLFunc() {
	type record struct {
		addr string
		ttl  time.Duration
	}

	// The looked-up data carries its own freshness.
	lookupFn := func(_ context.Context, _ string) (record, error) {
		return record{addr: "192.0.2.1", ttl: 30 * time.Second}, nil
	}

	// Each entry is cached for the TTL carried by its value instead of the
	// cache-wide default.
	c := sfcache.New(lookupFn, 8, 1*time.Minute,
		sfcache.WithTTLFunc(func(_ string, r record) time.Duration {
			return r.ttl
		}),
	)

	r, err := c.Lookup(context.TODO(), "example.com")

	fmt.Println(r.addr, err)

	// Output:
	// 192.0.2.1 <nil>
}

func ExampleWithStaleIfError() {
	var calls int

	// The first lookup succeeds, then the upstream becomes unavailable.
	lookupFn := func(_ context.Context, _ string) (string, error) {
		calls++

		if calls == 1 {
			return "value-1", nil
		}

		return "", errors.New("upstream outage")
	}

	// NOTE: WithStaleIfError requires explicit type instantiation matching
	// the cache types, because its argument does not mention them.
	c := sfcache.New(lookupFn, 8, 10*time.Millisecond,
		sfcache.WithStaleIfError[string, string](1*time.Minute),
	)

	val, err := c.Lookup(context.TODO(), "some_key")

	fmt.Println(val, err)

	time.Sleep(20 * time.Millisecond) // let the entry expire

	// The refresh fails: the last known good value is served instead.
	val, err = c.Lookup(context.TODO(), "some_key")

	fmt.Println(val, err)

	// Output:
	// value-1 <nil>
	// value-1 <nil>
}

func ExampleCache_PurgeExpired() {
	lookupFn := func(_ context.Context, key string) (string, error) {
		return key, nil
	}

	c := sfcache.New(lookupFn, 8, 10*time.Millisecond)

	_, _ = c.Lookup(context.TODO(), "some_key")

	time.Sleep(20 * time.Millisecond) // let the entry expire

	// Expired entries are otherwise only removed lazily.
	fmt.Println(c.PurgeExpired(), c.Len())

	// Output:
	// 1 0
}
