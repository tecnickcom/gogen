// Package dnscache provides a local DNS cache for LookupHost.
// The cache has a maximum size and a time-to-live (TTL) for each DNS entry.
package dnscache

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"
)

// Resolver is a net.Resolver interface for DNS lookups.
type Resolver interface {
	LookupHost(ctx context.Context, host string) (addrs []string, err error)
}

// entry represents a DNS cache entry for a host.
type entry struct {
	// wg wait for each duplicate lookup call for the same host.
	wg *sync.WaitGroup

	// err is the error returned by the external DNS lookup.
	err error

	// expireAt is the expiration time in seconds elapsed since January 1, 1970 UTC.
	expireAt int64

	// addrs is the list of IP addresses associated with the host by the DNS.
	addrs []string
}

// Cache represents a cache for DNS items.
type Cache struct {
	// hostmap maps a host name to a DNS item.
	hostmap map[string]*entry

	// resolver is the net.resolver used to resolve DNS queries.
	resolver Resolver

	// mux is the mutex for the cache.
	mux *sync.RWMutex

	// ttl is the time-to-live for DNS items.
	ttl time.Duration

	// size is the maximum size of the cache (min = 1).
	size int
}

// New creates a new DNS resolver with a cache of the specified size and TTL.
// If the resolver parameter is nil, a default resolver will be used.
// The size parameter determines the maximum number of DNS entries that can be cached (min = 1).
// If the size is less than or equal to zero, the cache will have a default size of 1.
// The ttl parameter specifies the time-to-live for each cached DNS entry.
func New(resolver Resolver, size int, ttl time.Duration) *Cache {
	if resolver == nil {
		resolver = &net.Resolver{}
	}

	if size <= 0 {
		size = 1
	}

	return &Cache{
		resolver: resolver,
		mux:      &sync.RWMutex{},
		ttl:      ttl,
		size:     size,
		hostmap:  make(map[string]*entry, size),
	}
}

// Reset clears the whole cache.
func (c *Cache) Reset() {
	c.mux.Lock()
	defer c.mux.Unlock()

	c.hostmap = make(map[string]*entry, c.size)
}

// Remove removes the cache entry for the specified host.
func (c *Cache) Remove(host string) {
	c.mux.Lock()
	defer c.mux.Unlock()

	delete(c.hostmap, host)
}

// LookupHost performs a DNS lookup for the given host using the DNSCacheResolver.
// It first checks if the host is already cached and not expired. If so, it returns
// the cached addresses. Otherwise, it performs a DNS lookup using the underlying
// Resolver and caches the obtained addresses for future use.
// Duplicate lookup calls for the same host will wait for the first lookup to complete.
//
//nolint:gocognit
func (c *Cache) LookupHost(ctx context.Context, host string) ([]string, error) {
	c.mux.Lock()
	item, ok := c.hostmap[host]

	if ok {
		if item.expireAt > time.Now().UTC().Unix() {
			c.mux.Unlock()
			return item.addrs, item.err
		}

		if item.wg != nil {
			// another external DNS lookup is already in progress,
			// waiting for completion and return values from cache.
			c.mux.Unlock()

			done := make(chan struct{})
			go func() {
				item.wg.Wait()
				close(done)
			}()

			select {
			case <-done:
			case <-ctx.Done():
				return nil, fmt.Errorf("context canceled: %w", ctx.Err())
			}

			c.mux.RLock()
			item, ok := c.hostmap[host]
			c.mux.RUnlock()

			if ok {
				return item.addrs, item.err
			}

			// the cache entry was removed during the wait,
			// move on to perform a new DNS lookup.
			c.mux.Lock()
		}
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)
	defer wg.Done()

	c.set(host, nil, nil, wg)
	c.mux.Unlock()

	addrs, err := c.resolver.LookupHost(ctx, host)

	c.mux.Lock()
	c.set(host, addrs, err, nil)
	c.mux.Unlock()

	return addrs, err //nolint:wrapcheck
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

// set adds or updates the cache entry for the given host with the provided addresses.
// If the cache is full, it will free up space by removing expired or old entries.
// If the host already exists in the cache, it will update the entry with the new addresses.
func (c *Cache) set(host string, addrs []string, err error, wg *sync.WaitGroup) {
	_, ok := c.hostmap[host]
	if (!ok) && (len(c.hostmap) >= c.size) {
		// free up space for a new entry
		c.evict()
	}

	var now int64

	if addrs != nil {
		now = time.Now().UTC().Add(c.ttl).Unix()
	}

	c.hostmap[host] = &entry{
		wg:       wg,
		err:      err,
		expireAt: now,
		addrs:    addrs,
	}
}

// evict removes either the oldest entry or the first expired one from the DNS cache.
// NOTE: this is not thread-safe, it should be called within a mutex lock.
func (c *Cache) evict() {
	cuttime := time.Now().UTC().Unix()
	oldest := int64(1<<63 - 1)
	oldestHost := ""

	for h, d := range c.hostmap {
		if d.expireAt < cuttime {
			delete(c.hostmap, h)
			return
		}

		if d.expireAt < oldest {
			oldest = d.expireAt
			oldestHost = h
		}
	}

	delete(c.hostmap, oldestHost)
}
