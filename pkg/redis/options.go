package redis

// Option customizes client configuration.
type Option func(*cfg)

// WithMessageEncodeFunc overrides the encoder used by SendData and SetData.
func WithMessageEncodeFunc(f TEncodeFunc) Option {
	return func(c *cfg) {
		c.messageEncodeFunc = f
	}
}

// WithMessageDecodeFunc overrides the decoder used by ReceiveData and GetData.
func WithMessageDecodeFunc(f TDecodeFunc) Option {
	return func(c *cfg) {
		c.messageDecodeFunc = f
	}
}

// WithSubscrChannels sets channels subscribed at client creation time.
func WithSubscrChannels(channels ...string) Option {
	return func(c *cfg) {
		c.subChannels = channels
	}
}

// WithSubscrChannelOptions sets subscription channel options for go-redis Pub/Sub.
func WithSubscrChannelOptions(opts ...ChannelOption) Option {
	return func(c *cfg) {
		c.subChannelOpts = opts
	}
}
