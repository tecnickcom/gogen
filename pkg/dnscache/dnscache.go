/*
Package dnscache provides a local DNS cache that is safe for concurrent use,
bounded in size, and uses single-flight request collapsing to avoid duplicate
lookups.

The package is designed as a drop-in complement to the standard net package.
It exposes LookupHost and DialContext helpers that cache resolved host names,
so repeated DNS lookups for the same host return cached results and only one
outstanding lookup is performed at a time.

Key features:

  - fixed-size in-memory cache with configurable capacity
  - TTL-based expiry (one cache-wide TTL for all entries) to keep DNS data
    fresh; authoritative DNS record TTLs are not consulted
  - thread-safe access for concurrent goroutines
  - single-flight behavior so duplicate lookups share one network request
  - optional custom net.Resolver support with sensible default behavior
  - DialContext helper that resolves host names and tries each returned IP until
    one connection succeeds

Why it matters:

  - reduces DNS resolution latency for repeated host names
  - lowers load on upstream resolvers and avoids query storms
  - keeps memory usage predictable with a capped entry count
  - makes DNS-heavy applications more resilient under concurrency
  - provides a practical http.Transport DialContext replacement for DNS-aware
    clients

Use this package in any Go service that performs frequent DNS lookups or needs
an efficient, safe cache for host resolution.
*/
package dnscache

import (
	"context"
	"fmt"
	"net"
	"slices"
	"time"

	"github.com/tecnickcom/gogen/pkg/sfcache"
)

// Resolver is a net.Resolver interface for DNS lookups.
type Resolver interface {
	LookupHost(ctx context.Context, host string) (addrs []string, err error)
}

// Cache provides DNS resolution caching with TTL and single-flight deduplication.
type Cache struct {
	cache *sfcache.Cache[string, []string]
}

// New creates a concurrent DNS cache with TTL expiry and single-flight lookups.
//
// If resolver is nil, a default net.Resolver is used. size bounds cache
// capacity (minimum effective size is 1), and ttl controls how long each
// hostname resolution remains valid.
//
// This constructor is useful for DNS-heavy clients that need lower latency and
// fewer duplicate upstream queries.
func New(resolver Resolver, size int, ttl time.Duration) *Cache {
	if resolver == nil {
		resolver = &net.Resolver{}
	}

	lookupFn := func(ctx context.Context, key string) ([]string, error) {
		addrs, err := resolver.LookupHost(ctx, key)
		if err != nil {
			return nil, fmt.Errorf("DNS lookup failed: %w", err)
		}

		return addrs, nil
	}

	return &Cache{
		cache: sfcache.New(lookupFn, size, ttl),
	}
}

// LookupHost resolves host to IP addresses using cache-first semantics.
//
// On cache miss or expiry, one goroutine performs the DNS lookup while other
// concurrent callers for the same host wait and share the result.
//
// This reduces resolver load and avoids thundering-herd lookups.
// The returned slice is a copy: callers may freely modify it without
// affecting the cached entry shared with other callers.
func (c *Cache) LookupHost(ctx context.Context, host string) ([]string, error) {
	val, err := c.cache.Lookup(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve DNS for host %s: %w", host, err)
	}

	// Return a copy: cached values are shared by reference across callers.
	return slices.Clone(val), nil
}

// DialContext resolves address through the cache and dials resolved IPs in order.
//
// It is intended as a drop-in replacement for transport DialContext functions
// (for example in http.Transport) when DNS caching is desired. The method tries
// each resolved IP until one connection succeeds.
func (c *Cache) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("failed to extract host and port from %s: %w", address, err)
	}

	ips, err := c.LookupHost(ctx, host)
	if err != nil {
		return nil, err
	}

	if len(ips) == 0 {
		return nil, fmt.Errorf("no addresses resolved for %s", host)
	}

	var (
		conn   net.Conn
		dialer net.Dialer
	)

	for _, ip := range ips {
		if net.ParseIP(ip) == nil {
			err = fmt.Errorf("invalid IP address: %s", ip)
			continue
		}

		conn, err = dialer.DialContext(ctx, network, net.JoinHostPort(ip, port))
		if err == nil {
			return conn, nil
		}
	}

	return nil, fmt.Errorf("failed to dial %s: %w", address, err)
}

// Len reports the current number of cached host entries.
func (c *Cache) Len() int {
	return c.cache.Len()
}

// Reset clears all cached DNS entries.
func (c *Cache) Reset() {
	c.cache.Reset()
}

// Remove evicts a single host entry from the cache.
func (c *Cache) Remove(host string) {
	c.cache.Remove(host)
}

// PurgeExpired removes all expired host entries from the cache and returns
// the number of entries removed. Expired entries are otherwise only removed
// lazily, when capacity pressure or a new lookup replaces them.
func (c *Cache) PurgeExpired() int {
	return c.cache.PurgeExpired()
}
