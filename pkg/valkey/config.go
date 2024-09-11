package valkey

import (
	"context"
	"errors"
	"regexp"
)

const (
	regexPatternHostPort = `^[^\:]*:[0-9]{2,5}$`
)

var regexHostPort = regexp.MustCompile(regexPatternHostPort)

type cfg struct {
	messageEncodeFunc TEncodeFunc
	messageDecodeFunc TDecodeFunc
	srvOpts           SrvOptions
	channels          []string
	vkclient          VKClient
}

func loadConfig(_ context.Context, srvOpts SrvOptions, opts ...Option) (*cfg, error) {
	c := &cfg{
		messageEncodeFunc: DefaultMessageEncodeFunc,
		messageDecodeFunc: DefaultMessageDecodeFunc,
		srvOpts:           srvOpts,
	}

	if (len(srvOpts.InitAddress) == 0) || (!regexHostPort.MatchString(srvOpts.InitAddress[0])) {
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
