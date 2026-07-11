package dnscache_test

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/tecnickcom/nurago/pkg/dnscache"
)

// staticResolver is a deterministic Resolver used to keep the examples
// reproducible; production code normally passes nil to use net.Resolver.
type staticResolver struct{}

func (staticResolver) LookupHost(_ context.Context, _ string) ([]string, error) {
	return []string{"192.0.2.1", "192.0.2.2"}, nil
}

func ExampleNew() {
	// Create a DNS cache holding up to 128 hosts for 5 minutes each,
	// using the default net.Resolver.
	cache := dnscache.New(nil, 128, 5*time.Minute)

	// Wire the cache into an http.Transport so every request reuses cached
	// DNS resolutions instead of querying the resolver again.
	client := &http.Client{
		Transport: &http.Transport{
			DialContext: cache.DialContext,
		},
	}

	_ = client // use the client as usual

	fmt.Println(cache.Len())

	// Output: 0
}

func ExampleCache_LookupHost() {
	cache := dnscache.New(staticResolver{}, 128, 5*time.Minute)

	// The first call queries the resolver; repeated calls within the TTL are
	// served from the cache, with concurrent lookups for the same host
	// coalesced into a single upstream query.
	addrs, err := cache.LookupHost(context.Background(), "example.com")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(addrs)
	fmt.Println(cache.Len())

	// Output:
	// [192.0.2.1 192.0.2.2]
	// 1
}
