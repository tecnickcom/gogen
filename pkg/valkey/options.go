package valkey

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

// WithValkeyClient injects an existing Valkey client, primarily for testing.
func WithValkeyClient(vkclient VKClient) Option {
	return func(c *cfg) {
		c.vkclient = vkclient
	}
}
