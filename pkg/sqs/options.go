package sqs

import (
	"context"
	"net/url"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sep "github.com/aws/smithy-go/endpoints"
	"github.com/tecnickcom/gogen/pkg/awsopt"
)

// SrvOptionFunc aliases an AWS SDK SQS service option mutator.
type SrvOptionFunc = func(*sqs.Options)

// Option applies a configuration change to the internal SQS client settings.
type Option func(*cfg)

// WithAWSOptions appends awsopt options used to build aws.Config.
func WithAWSOptions(opt awsopt.Options) Option {
	return func(c *cfg) {
		c.awsOpts = append(c.awsOpts, opt...)
	}
}

// WithSrvOptionFuncs appends service-level sqs.Options mutators.
func WithSrvOptionFuncs(opt ...SrvOptionFunc) Option {
	return func(c *cfg) {
		c.srvOptFns = append(c.srvOptFns, opt...)
	}
}

// WithEndpointMutable sets BaseEndpoint while preserving SDK endpoint mutability.
func WithEndpointMutable(url string) Option {
	return WithSrvOptionFuncs(
		func(o *sqs.Options) {
			o.BaseEndpoint = aws.String(url)
		},
	)
}

// WithEndpointImmutable installs a fixed EndpointResolverV2 for deterministic endpoint routing.
func WithEndpointImmutable(url string) Option {
	return WithSrvOptionFuncs(
		func(o *sqs.Options) {
			o.EndpointResolverV2 = &endpointResolver{url: url}
		},
	)
}

// endpointResolver resolves all SQS requests to a fixed endpoint URL.
type endpointResolver struct {
	url string
}

// ResolveEndpoint parses and returns the configured fixed endpoint URL.
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

// WithWaitTimeSeconds sets long-poll wait duration in seconds for ReceiveMessage calls.
// Values range: 0 to 20 seconds.
func WithWaitTimeSeconds(t int32) Option {
	return func(c *cfg) {
		c.waitTimeSeconds = t
	}
}

// WithVisibilityTimeout sets message invisibility duration in seconds after receive.
// Values range: 0 to 43200. Maximum: 12 hours.
func WithVisibilityTimeout(t int32) Option {
	return func(c *cfg) {
		c.visibilityTimeout = t
	}
}

// WithMessageEncodeFunc overrides the serializer used by SendData.
func WithMessageEncodeFunc(f TEncodeFunc) Option {
	return func(c *cfg) {
		c.messageEncodeFunc = f
	}
}

// WithMessageDecodeFunc overrides the deserializer used by ReceiveData.
// The data argument passed to ReceiveData must be a pointer to the expected type.
func WithMessageDecodeFunc(f TDecodeFunc) Option {
	return func(c *cfg) {
		c.messageDecodeFunc = f
	}
}
