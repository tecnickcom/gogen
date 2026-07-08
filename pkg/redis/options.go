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

// WithChannels sets Pub/Sub channels subscribed at client creation time.
func WithChannels(channels ...string) Option {
	return func(c *cfg) {
		c.channels = channels
	}
}

// WithChannelOptions sets subscription channel options for go-redis Pub/Sub
// (e.g. libredis.WithChannelSize, libredis.WithChannelSendTimeout,
// libredis.WithChannelHealthCheckInterval). It has no effect unless
// WithChannels configures at least one channel.
func WithChannelOptions(opts ...ChannelOption) Option {
	return func(c *cfg) {
		c.channelOpts = opts
	}
}

// WithRedisClient injects an existing go-redis client, primarily for testing.
//
// When a client is injected, the server options passed to New are never
// consulted and their address is not validated. Injection is intended for
// command-path testing: Get, Set, Del, Publish, and Ping results can be built
// with the public go-redis constructors (libredis.NewStringResult and
// friends). Combining an injected client with WithChannels requires its
// Subscribe to return a functional *libredis.PubSub, so subscription flows
// are better tested against a real server.
func WithRedisClient(rclient RClient) Option {
	return func(c *cfg) {
		c.rclient = rclient
	}
}
