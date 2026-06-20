package redis

import (
	"errors"
	"net"
)

// cfg defines the redis client configuration.
type cfg struct {
	messageEncodeFunc TEncodeFunc
	messageDecodeFunc TDecodeFunc
	srvOpts           *SrvOptions
	subChannels       []string
	subChannelOpts    []ChannelOption
}

// loadConfig loads and validates the redis client configuration.
func loadConfig(srvOpts *SrvOptions, opts ...Option) (*cfg, error) {
	c := &cfg{
		messageEncodeFunc: DefaultMessageEncodeFunc,
		messageDecodeFunc: DefaultMessageDecodeFunc,
		srvOpts:           srvOpts,
	}

	if !validAddr(srvOpts) {
		return nil, errors.New("missing or invalid redis client options")
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

// validAddr reports whether the server options carry a syntactically valid
// host:port address. The address parsing is delegated to net.SplitHostPort,
// which also accepts bracketed IPv6 hosts (e.g. "[::1]:6379"); go-redis
// performs the actual connection-time validation.
func validAddr(srvOpts *SrvOptions) bool {
	if srvOpts == nil {
		return false
	}

	_, port, err := net.SplitHostPort(srvOpts.Addr)
	if err != nil {
		return false
	}

	return port != ""
}
