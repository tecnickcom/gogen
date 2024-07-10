package redis

// Option is a type to allow setting custom client options.
type Option func(*cfg)

// WithMessageEncodeFunc allow to replace DefaultMessageEncodeFunc.
// This function used by SendData() to encode and serialize the input data to a string compatible with SQS.
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

// WithSubscrChannels sets the channels to subscribe to and receive data from.
func WithSubscrChannels(channels ...string) Option {
	return func(c *cfg) {
		c.subChannels = channels
	}
}

// WithSubscrChannelOptions sets options for the subscribed channels.
func WithSubscrChannelOptions(opts ...ChannelOption) Option {
	return func(c *cfg) {
		c.subChannelOpts = opts
	}
}
