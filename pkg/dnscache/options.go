package dnscache

import (
	"net"
	"time"
)

// config accumulates optional settings applied by [Option] values before a
// [Cache] is built by [New].
type config struct {
	dialer            *net.Dialer
	dialTimeout       time.Duration
	maxStale          time.Duration
	maxStaleOnFailure time.Duration
	rotate            bool
}

// Option customizes a [Cache] created by [New].
type Option func(*config)

// WithDialer sets the dialer used by [Cache.DialContext], configuring the connection
// timeout, keep-alive, local address, or a Control hook. A nil dialer is ignored,
// leaving the default (keep-alive of 30s, no explicit timeout) in place.
//
// A non-zero dialer Timeout bounds each address attempt just like [WithDialTimeout];
// when both are configured, the shorter bound wins.
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
// [Cache.DialContext] may take, so a single unresponsive address cannot consume the
// whole caller deadline before the remaining addresses are tried. It is layered on top
// of the caller's context; a value <= 0 (the default) disables it.
//
// NOTE: a value shorter than a legitimately slow connect will abort working
// high-latency links. A Timeout on the dialer passed to [WithDialer] has the same
// per-attempt effect; when both are configured, the shorter bound wins.
func WithDialTimeout(timeout time.Duration) Option {
	return func(cfg *config) {
		cfg.dialTimeout = timeout
	}
}

// WithAddressRotation rotates the order in which resolved addresses are dialed on each
// [Cache.DialContext] call, spreading connections across a host's records instead of
// always trying the resolver's first address. It is disabled by default, so the
// resolver's RFC 6724 address ordering is preserved.
//
// Rotation is applied within each address family, so the resolver-preferred family
// always stays first in the dial order.
func WithAddressRotation() Option {
	return func(cfg *config) {
		cfg.rotate = true
	}
}

// WithStaleIfError serves the last successfully resolved addresses when a later
// refresh fails, until maxStale past their original expiry (RFC 5861
// stale-if-error, see
// [github.com/tecnickcom/nurago/pkg/sfcache.Config.MaxStale]).
// A maxStale <= 0 disables it (the default).
//
// PRECONDITION: because the window is anchored to the expiry and not to the
// failure, only a host resolved more often than ttl + maxStale is protected. A
// host that has been idle for longer than ttl + maxStale has no stale
// protection at all: the resolver error is returned even though its last known
// good addresses are still cached. Use [WithStaleOnFailure] to protect
// rarely resolved hosts too.
func WithStaleIfError(maxStale time.Duration) Option {
	return func(cfg *config) {
		cfg.maxStale = maxStale
	}
}

// WithStaleOnFailure serves the last successfully resolved addresses for up to
// maxStaleOnFailure after a refresh first fails, however long the host had been
// idle before the failure (see
// [github.com/tecnickcom/nurago/pkg/sfcache.Config.MaxStaleOnFailure]).
//
// Unlike [WithStaleIfError], it also protects hosts that are resolved rarely. The
// window is anchored once, by the first failed refresh: further failures keep serving
// the same addresses until that deadline but never push it back, so a broken resolver
// cannot pin an address set forever. Every call still attempts a fresh resolution and
// recovery is automatic on the first success.
//
// A maxStaleOnFailure <= 0 disables it (the default). When both options are
// set, the addresses are served stale until the later of the two deadlines.
func WithStaleOnFailure(maxStaleOnFailure time.Duration) Option {
	return func(cfg *config) {
		cfg.maxStaleOnFailure = maxStaleOnFailure
	}
}
