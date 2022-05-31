package sqs

import (
	"github.com/nexmoinc/gosrvlib/pkg/awsopt"
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
