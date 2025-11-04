package sqs

import (
	"context"
	"net/url"

	"github.com/Vonage/gosrvlib/pkg/awsopt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sep "github.com/aws/smithy-go/endpoints"
)

// SrvOptionFunc is an alias for this service option function.
type SrvOptionFunc = func(*sqs.Options)

// Option is a type to allow setting custom client options.
type Option func(*cfg)

// WithAWSOptions allows to add an arbitrary AWS options.
func WithAWSOptions(opt awsopt.Options) Option {
	return func(c *cfg) {
		c.awsOpts = append(c.awsOpts, opt...)
	}
}

// WithSrvOptionFuncs allows to specify specific options.
func WithSrvOptionFuncs(opt ...SrvOptionFunc) Option {
	return func(c *cfg) {
		c.srvOptFns = append(c.srvOptFns, opt...)
	}
}

// WithEndpointMutable sets a mutable endpoint.
func WithEndpointMutable(url string) Option {
	return WithSrvOptionFuncs(
		func(o *sqs.Options) {
			o.BaseEndpoint = aws.String(url)
		},
	)
}

// WithEndpointImmutable sets an immutable endpoint.
func WithEndpointImmutable(url string) Option {
	return WithSrvOptionFuncs(
		func(o *sqs.Options) {
			o.EndpointResolverV2 = &endpointResolver{url: url}
		},
	)
}

type endpointResolver struct {
	url string
}

func (r *endpointResolver) ResolveEndpoint(_ context.Context, _ sqs.EndpointParameters) (
	sep.Endpoint,
	error,
) {
	u, err := url.Parse(r.url)
	if err != nil {
		return sep.Endpoint{}, err //nolint:wrapcheck
	}

	return sep.Endpoint{URI: *u}, nil
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
