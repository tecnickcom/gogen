package valkey

import (
	"context"
	"errors"
	"net"
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
func loadConfig(_ context.Context, srvOpts SrvOptions, opts ...Option) (*cfg, error) {
	c := &cfg{
		messageEncodeFunc: DefaultMessageEncodeFunc,
		messageDecodeFunc: DefaultMessageDecodeFunc,
		srvOpts:           srvOpts,
	}

	if !validInitAddress(srvOpts.InitAddress) {
		return nil, errors.New("missing or invalid valkey client options")
	}

	for _, apply := range opts {
		apply(c)
	}

	if c.messageEncodeFunc == nil {
		return nil, errors.New("missing message encoding function")
	}

	if c.messageDecodeFunc == nil {
		return nil, errors.New("missing message decoding function")
	}

	return c, nil
}
