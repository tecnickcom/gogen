package valkey

// Option is a type to allow setting custom client options.
type Option func(*cfg)

// WithMessageEncodeFunc allow to replace DefaultMessageEncodeFunc.
// This function used by SendData() to encode and serialize the input data to a string.
func WithMessageEncodeFunc(f TEncodeFunc) Option {
	return func(c *cfg) {
		c.messageEncodeFunc = f
	}
}

// WithMessageDecodeFunc allow to replace DefaultMessageDecodeFunc().
// This function used by ReceiveData() to decode a message encoded with messageEncodeFunc to the provided data object.
// The value underlying data must be a pointer to the correct type for the next data item received.
func WithMessageDecodeFunc(f TDecodeFunc) Option {
	return func(c *cfg) {
		c.messageDecodeFunc = f
	}
}

// WithChannels sets the channels to subscribe to and receive data from.
func WithChannels(channels ...string) Option {
	return func(c *cfg) {
		c.channels = channels
	}
}

// WithValkeyClient overrides the default Valkey client.
// This function is mainly used for testing.
func WithValkeyClient(vkclient VKClient) Option {
	return func(c *cfg) {
		c.vkclient = vkclient
	}
}
