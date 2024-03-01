// Package dnscache provides a local DNS cache for LookupHost.
// The cache has a maximum size and a time-to-live (TTL) for each DNS entry.
package dnscache

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/Vonage/gosrvlib/pkg/threadsafe/tsmap"
)

// dnsItem represents a DNS cache entry for a host.
type dnsItem struct {
	// expireAt is the expiration time in seconds elapsed since January 1, 1970 UTC.
	expireAt int64

	// addrs is the list of IP addresses associated with the host by the DNS.
	addrs []string
}

// Resolver is a net.Resolver interface for DNS lookups.
type Resolver interface {
	LookupHost(ctx context.Context, host string) (addrs []string, err error)
}

// CacheResolver represents a cache for DNS items.
type CacheResolver struct {
	// resolver is the net.resolver used to resolve DNS queries.
	resolver Resolver

	// mux is the mutex for the cache.
	mux *sync.RWMutex

	// ttl is the time-to-live for DNS items.
	ttl time.Duration

	// size is the maximum size of the cache (min = 1).
	size int

	// cache maps a host name to a DNS item.
	cache map[string]*dnsItem
}

// New creates a new DNS resolver with a cache of the specified size and TTL.
// If the resolver parameter is nil, a default resolver will be used.
// The size parameter determines the maximum number of DNS entries that can be cached (min = 1).
// If the size is less than or equal to zero, the cache will have a default size of 1.
// The ttl parameter specifies the time-to-live for each cached DNS entry.
func New(resolver Resolver, size int, ttl time.Duration) *CacheResolver {
	if resolver == nil {
		resolver = &net.Resolver{}
	}

	if size <= 0 {
		size = 1
	}

	return &CacheResolver{
		resolver: resolver,
		mux:      &sync.RWMutex{},
		ttl:      ttl,
		size:     size,
		cache:    make(map[string]*dnsItem, size),
	}
}

// Reset clears the whole cache and initializes it with a new map of the specified size.
func (r *CacheResolver) Reset() {
	r.mux.Lock()
	defer r.mux.Unlock()

	r.cache = make(map[string]*dnsItem, r.size)
}

// RemoveEntry removes the cache entry for the specified host.
func (r *CacheResolver) RemoveEntry(host string) {
	tsmap.Delete(r.mux, r.cache, host)
}

// LookupHost performs a DNS lookup for the given host using the DNSCacheResolver.
// It first checks if the host is already cached and not expired. If so, it returns
// the cached addresses. Otherwise, it performs a DNS lookup using the underlying
// Resolver and caches the obtained addresses for future use.
func (r *CacheResolver) LookupHost(ctx context.Context, host string) ([]string, error) {
	item, exist := tsmap.GetOK(r.mux, r.cache, host)
	if exist && (item.expireAt > time.Now().UTC().Unix()) {
		return item.addrs, nil
	}

	addrs, err := r.resolver.LookupHost(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("failed DNS lookup for the host %s : %w", host, err)
	}

	r.set(host, addrs, exist)

	return addrs, nil
}

// DialContext dials the network and address specified by the parameters.
// It resolves the host from the address using the LookupHost method of the Resolver.
// It then attempts to establish a connection to each resolved IP address until a successful connection is made.
// If all connection attempts fail, it returns an error.
// The function returns the established net.Conn and any error encountered during the process.
// This function can replace the DialContext in http.Transport.
func (r *CacheResolver) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("failed to extract host and port from %s: %w", address, err)
	}

	ips, err := r.LookupHost(ctx, host)
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
func (r *CacheResolver) set(host string, addrs []string, exist bool) {
	if (!exist) && (len(r.cache) >= r.size) {
		r.evict()
	}

	tsmap.Set(
		r.mux,
		r.cache,
		host,
		&dnsItem{
			expireAt: time.Now().UTC().Add(r.ttl).Unix(),
			addrs:    addrs,
		},
	)
}

// evict removes either the oldest entry or the first expired one from the DNS cache.
func (r *CacheResolver) evict() {
	cuttime := time.Now().UTC().Unix()
	oldest := int64(1<<63 - 1)
	oldestHost := ""

	for h, d := range r.cache {
		if d.expireAt < cuttime {
			r.RemoveEntry(h)
			return
		}

		if d.expireAt < oldest {
			oldest = d.expireAt
			oldestHost = h
		}
	}

	r.RemoveEntry(oldestHost)
}
