package sqs

import (
	"github.com/Vonage/gosrvlib/pkg/awsopt"
)

// Option is a type to allow setting custom client options.
type Option func(*cfg)

// WithAWSOptions allows to add an arbitrary AWS options.
func WithAWSOptions(opt awsopt.Options) Option {
	return func(c *cfg) {
		c.awsOpts = append(c.awsOpts, opt...)
	}
}

// WithWaitTimeSeconds overrides the default duration (in seconds) for which the call waits for a message to arrive in the queue before returning.
// Values range: 0 to 20 seconds.
func WithWaitTimeSeconds(t int32) Option {
	return func(c *cfg) {
		c.waitTimeSeconds = t
	}
}

// WithVisibilityTimeout overrides the default duration (in seconds) that the received messages are hidden from subsequent retrieve requests after being retrieved by a ReceiveMessage request.
// Values range: 0 to 43200. Maximum: 12 hours.
func WithVisibilityTimeout(t int32) Option {
	return func(c *cfg) {
		c.visibilityTimeout = t
	}
}

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
