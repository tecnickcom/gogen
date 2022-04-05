package sqs

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

// Option is a type to allow setting custom client options.
type Option func(*cfg)

// WithWaitTimeSeconds overrides the default duration (in seconds) for which the call waits for a message to arrive in the queue before returning.
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

// WithEndpoint overrides the AWS endpoint for the service.
func WithEndpoint(url string, isImmutable bool) Option {
	return withAWSOption(config.WithEndpointResolverWithOptions(&endpointResolver{
		url:         url,
		isImmutable: isImmutable,
	}))
}

func withAWSOption(opt func(*config.LoadOptions) error) Option {
	return func(c *cfg) {
		c.awsOpts = append(c.awsOpts, opt)
	}
}

type endpointResolver struct {
	url         string
	isImmutable bool
}

func (r endpointResolver) ResolveEndpoint(_, region string, _ ...interface{}) (aws.Endpoint, error) {
	return aws.Endpoint{
		SigningRegion:     region,
		URL:               r.url,
		HostnameImmutable: r.isImmutable,
	}, nil
}
