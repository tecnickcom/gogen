/*
Package dnscache implements github.com/Vonage/gosrvlib/pkg/sfcache to provide a
simple, local, thread-safe, fixed-size, and single-flight cache for DNS lookup
calls.

This package provides LookupHost() and DialContext() functions that can be used
in place of the standard ones in the net package.

By caching previous values, dnscache improves DNS lookup performance by
eliminating the need for repeated expensive requests.

This package provides a local in-memory cache with a configurable maximum number
of entries. The fixed size helps with efficient memory management and prevents
excessive memory usage. The cache is thread-safe, allowing concurrent access
without the need for external synchronization. It efficiently handles concurrent
requests by sharing results from the first lookup, ensuring only one request
does the expensive call, and avoiding unnecessary network load or resource
starvation. Duplicate calls for the same key will wait for the first call to
complete and return the same value.

Each cache entry has a set time-to-live (TTL), so it will automatically expire.
However, it is also possible to force the removal of a specific DNS entry or
reset the entire cache.

This package is ideal for any Go application that relies heavily on DNS lookups.
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
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve DNS for host %s: %w", host, err)
	}

	return val.([]string), nil //nolint:forcetypeassert
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

// Len returns the number of items in the cache.
func (c *Cache) Len() int {
	return c.cache.Len()
}

// Reset clears the whole cache.
func (c *Cache) Reset() {
	c.cache.Reset()
}

// Remove removes the cache entry for the specified host.
func (c *Cache) Remove(host string) {
	c.cache.Remove(host)
}
