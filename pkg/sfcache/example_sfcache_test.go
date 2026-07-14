package sfcache_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/tecnickcom/nurago/pkg/sfcache"
)

func ExampleCache_Lookup() {
	// example lookup function that returns the key as value:
	// the cache value type V is inferred from its return type.
	lookupFn := func(_ context.Context, key string) (string, error) {
		return key, nil
	}

	// create a new cache with a lookupFn function, a maximum number of 3 entries, and a TTL of 1 minute.
	c := sfcache.New(lookupFn, sfcache.Config{Size: 3, TTL: 1 * time.Minute})

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
	c := sfcache.New(lookupFn, sfcache.Config{Size: 8, TTL: 1 * time.Minute},
		sfcache.WithTTLFunc(func(_ string, r record) time.Duration {
			return r.ttl
		}),
	)

	r, err := c.Lookup(context.TODO(), "example.com")

	fmt.Println(r.addr, err)

	// Output:
	// 192.0.2.1 <nil>
}

// outageLookupFn returns a lookup function that succeeds once and then behaves
// as if the upstream had gone down.
func outageLookupFn() func(context.Context, string) (string, error) {
	var calls int

	return func(_ context.Context, _ string) (string, error) {
		calls++

		if calls == 1 {
			return "value-1", nil
		}

		return "", errors.New("upstream outage")
	}
}

// ExampleNew_staleIfError shows the RFC 5861 stale window: it is measured from
// the value's original expiration, so it only covers a key that is looked up
// again within TTL + MaxStale.
func ExampleNew_staleIfError() {
	c := sfcache.New(outageLookupFn(), sfcache.Config{
		Size:     8,
		TTL:      10 * time.Millisecond,
		MaxStale: 1 * time.Minute,
	})

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

// ExampleNew_staleOnFailure shows the failure-anchored stale window: it is
// measured from the failed refresh, so it also covers a key that has been idle
// for far longer than TTL + MaxStale, which is where MaxStale alone would
// return the outage error instead.
func ExampleNew_staleOnFailure() {
	c := sfcache.New(outageLookupFn(), sfcache.Config{
		Size:              8,
		TTL:               10 * time.Millisecond,
		MaxStale:          10 * time.Millisecond,
		MaxStaleOnFailure: 1 * time.Minute,
	})

	val, err := c.Lookup(context.TODO(), "some_key")

	fmt.Println(val, err)

	// Idle well past TTL + MaxStale: the RFC 5861 window is long closed.
	time.Sleep(50 * time.Millisecond)

	// The refresh fails: the last known good value is served anyway, for
	// MaxStaleOnFailure measured from this failure.
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

	c := sfcache.New(lookupFn, sfcache.Config{Size: 8, TTL: 10 * time.Millisecond})

	_, _ = c.Lookup(context.TODO(), "some_key")

	time.Sleep(20 * time.Millisecond) // let the entry expire

	// Expired entries are otherwise only removed lazily.
	fmt.Println(c.PurgeExpired(), c.Len())

	// Output:
	// 1 0
}
