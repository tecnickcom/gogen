package dnscache

import (
	"net"
	"time"

	"github.com/tecnickcom/gogen/pkg/sfcache"
)

// config accumulates optional settings applied by [Option] values before a
// [Cache] is built by [New].
type config struct {
	dialer      *net.Dialer
	sfcacheOpts []sfcache.Option[string, []string]
	dialTimeout time.Duration
	rotate      bool
}

// Option customizes a [Cache] created by [New].
type Option func(*config)

// WithDialer sets the dialer used by [Cache.DialContext]. A nil dialer is
// ignored, leaving the default (keep-alive of 30s, no explicit timeout) in
// place. Use this to configure connection timeout, keep-alive, local address,
// or a Control hook, mirroring how net/http configures its transport dialer.
// A non-zero dialer Timeout bounds each address attempt just like
// [WithDialTimeout]; when both are configured, the shorter bound wins.
//
// The dialer's Resolver field is unused: addresses are pre-resolved by the
// cache's [Resolver] (configured via [New]), and the dialer only ever receives
// IP literals.
func WithDialer(dialer *net.Dialer) Option {
	return func(cfg *config) {
		if dialer != nil {
			cfg.dialer = dialer
		}
	}
}

// WithDialTimeout bounds how long each individual address dial attempt in
// [Cache.DialContext] may take, so a single unresponsive address cannot consume
// the whole caller deadline before the remaining addresses are tried. It is a
// per-attempt timeout layered on top of the caller's context; a value <= 0 (the
// default) disables it, leaving only the caller's context in force. Set it
// generously: a value shorter than a legitimately slow connect will abort
// working high-latency links. A Timeout on the dialer passed to [WithDialer]
// has the same per-attempt effect; when both are configured, the shorter
// bound wins.
func WithDialTimeout(timeout time.Duration) Option {
	return func(cfg *config) {
		cfg.dialTimeout = timeout
	}
}

// WithAddressRotation rotates the order in which resolved addresses are dialed
// on each [Cache.DialContext] call, spreading connections across a host's
// records instead of always trying the resolver's first address. It is disabled
// by default so the resolver's RFC 6724 address ordering is preserved; enable it
// when client-side load spreading across equivalent records is desired.
// Rotation is applied within each address family, so the resolver-preferred
// family always stays first in the dial order.
func WithAddressRotation() Option {
	return func(cfg *config) {
		cfg.rotate = true
	}
}

// WithStaleIfError serves the last successfully resolved addresses when a
// later refresh fails, until maxStale past their original expiry (see
// [sfcache.WithStaleIfError]). This keeps DNS-dependent clients working through
// transient resolver outages. A maxStale <= 0 disables it (the default).
func WithStaleIfError(maxStale time.Duration) Option {
	return func(cfg *config) {
		cfg.sfcacheOpts = append(cfg.sfcacheOpts, sfcache.WithStaleIfError[string, []string](maxStale))
	}
}
