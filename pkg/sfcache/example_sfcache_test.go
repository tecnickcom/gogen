package sfcache_test

import (
	"context"
	"fmt"
	"time"

	"github.com/Vonage/gosrvlib/pkg/sfcache"
)

func ExampleCache_Lookup() {
	// example lookup function that returns the key as value.
	lookupFn := func(_ context.Context, key string) (any, error) {
		return key, nil
	}

	// create a new cache with a lookupFn function, a maximum number of 3 entries, and a TTL of 1 minute.
	c := sfcache.New(lookupFn, 3, 1*time.Minute)

	val, err := c.Lookup(context.TODO(), "some_key")

	fmt.Println(val, err)

	// Output:
	// some_key <nil>
}
