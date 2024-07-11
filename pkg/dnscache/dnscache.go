/*
Package dnscache provides a thread-safe local single-flight DNS cache for LookupHost.
The cache has a maximum size and a time-to-live (TTL) for each DNS entry.
Duplicate LookupHost calls for the same host will wait for the first lookup to complete and return the same value.
*/
package dnscache

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/Vonage/gosrvlib/pkg/sfcache"
)

// Resolver is a net.Resolver interface for DNS lookups.
type Resolver interface {
	LookupHost(ctx context.Context, host string) (addrs []string, err error)
}

// Cache represents the single-flight DNS cache.
type Cache struct {
	cache *sfcache.Cache
}

// New creates a new single-flight DNS cache of the specified size and TTL.
// If the resolver parameter is nil, a default net.Resolver will be used.
// The size parameter determines the maximum number of DNS entries that can be cached (min = 1).
// If the size is less than or equal to zero, the cache will have a default size of 1.
// The ttl parameter specifies the time-to-live for each cached DNS entry.
func New(resolver Resolver, size int, ttl time.Duration) *Cache {
	if resolver == nil {
		resolver = &net.Resolver{}
	}

	lookupFn := func(ctx context.Context, key string) (any, error) {
		return resolver.LookupHost(ctx, key)
	}

	return &Cache{
		cache: sfcache.New(lookupFn, size, ttl),
	}
}

// LookupHost performs a DNS lookup for the given host.
// Duplicate lookup calls for the same host will wait for the first lookup to complete (single-flight).
// It also handles the case where the cache entry is removed or updated during the wait.
// The function returns the cached value if available; otherwise, it performs a new lookup.
// If the external lookup call is successful, it updates the cache with the newly obtained value.
func (c *Cache) LookupHost(ctx context.Context, host string) ([]string, error) {
	val, err := c.cache.Lookup(ctx, host)
	return val.([]string), err //nolint:forcetypeassert,wrapcheck
}

// DialContext dials the network and address specified by the parameters.
// It resolves the host from the address using the LookupHost method of the Resolver.
// It then attempts to establish a connection to each resolved IP address until a successful connection is made.
// If all connection attempts fail, it returns an error.
// The function returns the established net.Conn and any error encountered during the process.
// This function can replace the DialContext in http.Transport.
func (c *Cache) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("failed to extract host and port from %s: %w", address, err)
	}

	ips, err := c.LookupHost(ctx, host)
	if err != nil {
		return nil, err
	}

	var (
		conn   net.Conn
		dialer net.Dialer
	)

	for _, ip := range ips {
		conn, err = dialer.DialContext(ctx, network, net.JoinHostPort(ip, port))
		if err == nil {
			return conn, nil
		}
	}

	return nil, fmt.Errorf("failed to dial %s: %w", address, err)
}
