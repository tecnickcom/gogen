package redis

import (
	"net"
	"slices"
	"strings"
)

// cfg defines the redis client configuration.
type cfg struct {
	messageEncodeFunc TEncodeFunc
	messageDecodeFunc TDecodeFunc
	srvOpts           *SrvOptions
	channels          []string
	channelOpts       []ChannelOption
	rclient           RClient
}

// loadConfig loads and validates the redis client configuration.
func loadConfig(srvOpts *SrvOptions, opts ...Option) (*cfg, error) {
	c := &cfg{
		messageEncodeFunc: DefaultMessageEncodeFunc,
		messageDecodeFunc: DefaultMessageDecodeFunc,
		srvOpts:           srvOpts,
	}

	for _, apply := range opts {
		apply(c)
	}

	// The server options are only used to dial a new client. When a client is
	// injected via WithRedisClient they are never consulted, so they are
	// validated only when the package would actually connect.
	if c.rclient == nil && !validAddr(srvOpts) {
		return nil, ErrInvalidOptions
	}

	// An empty channel name builds a protocol-legal SUBSCRIBE to the
	// empty-string channel, which is almost certainly a caller bug.
	if slices.Contains(c.channels, "") {
		return nil, ErrInvalidChannelName
	}

	if c.messageEncodeFunc == nil {
		return nil, ErrNilEncodeFunc
	}

	if c.messageDecodeFunc == nil {
		return nil, ErrNilDecodeFunc
	}

	return c, nil
}

// validAddr reports whether the server options carry a usable address.
//
// Unix domain socket configurations only require a non-empty Addr: either
// Network is set to "unix", or Network is unset and Addr starts with "/"
// (which go-redis then auto-detects as unix). With any other Network the
// path shortcut does not apply, since go-redis would dial that network with
// the path verbatim. TCP addresses must parse as host:port via
// net.SplitHostPort, which also accepts bracketed IPv6 hosts (e.g.
// "[::1]:6379") and an omitted host (":6379", dialed as localhost); go-redis
// performs the actual connection-time validation.
func validAddr(srvOpts *SrvOptions) bool {
	if srvOpts == nil {
		return false
	}

	if (srvOpts.Network == "unix") || ((srvOpts.Network == "") && strings.HasPrefix(srvOpts.Addr, "/")) {
		return srvOpts.Addr != ""
	}

	_, port, err := net.SplitHostPort(srvOpts.Addr)
	if err != nil {
		return false
	}

	return port != ""
}
