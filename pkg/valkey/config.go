package valkey

import (
	"net"
	"slices"
)

// validInitAddress reports whether every entry in addrs is a non-empty
// host:port pair. The valkey-go client performs the authoritative address
// validation on connect, so this only rejects obviously malformed entries
// (empty list, missing host, or missing port) while accepting IPv6 addresses
// and single-digit ports.
func validInitAddress(addrs []string) bool {
	if len(addrs) == 0 {
		return false
	}

	for _, addr := range addrs {
		host, port, err := net.SplitHostPort(addr)
		if (err != nil) || (host == "") || (port == "") {
			return false
		}
	}

	return true
}

// cfg holds the configuration for the valkey client.
type cfg struct {
	messageEncodeFunc TEncodeFunc
	messageDecodeFunc TDecodeFunc
	srvOpts           SrvOptions
	channels          []string
	vkclient          VKClient
}

// loadConfig loads and validates the configuration for the valkey client.
func loadConfig(srvOpts SrvOptions, opts ...Option) (*cfg, error) {
	c := &cfg{
		messageEncodeFunc: DefaultMessageEncodeFunc,
		messageDecodeFunc: DefaultMessageDecodeFunc,
		srvOpts:           srvOpts,
	}

	for _, apply := range opts {
		apply(c)
	}

	// The server address is only used to dial a new client. When a client is
	// injected via WithValkeyClient the address is never consulted, so it is
	// validated only when the package would actually connect.
	if c.vkclient == nil && !validInitAddress(srvOpts.InitAddress) {
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
