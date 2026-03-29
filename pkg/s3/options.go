package s3

import (
	"context"
	"net/url"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	sep "github.com/aws/smithy-go/endpoints"
	"github.com/tecnickcom/gogen/pkg/awsopt"
)

// SrvOptionFunc aliases an AWS SDK S3 service option mutator.
type SrvOptionFunc = func(*s3.Options)

// Option applies a configuration change to the internal S3 client settings.
type Option func(*cfg)

// WithAWSOptions appends awsopt options used to build aws.Config.
func WithAWSOptions(opt awsopt.Options) Option {
	return func(c *cfg) {
		c.awsOpts = append(c.awsOpts, opt...)
	}
}

// WithSrvOptionFuncs appends service-specific s3.Options mutators.
func WithSrvOptionFuncs(opt ...SrvOptionFunc) Option {
	return func(c *cfg) {
		c.srvOptFns = append(c.srvOptFns, opt...)
	}
}

// WithEndpointMutable sets BaseEndpoint while allowing SDK endpoint behavior to remain mutable.
func WithEndpointMutable(url string) Option {
	return WithSrvOptionFuncs(
		func(o *s3.Options) {
			o.BaseEndpoint = aws.String(url)
		},
	)
}

// WithEndpointImmutable installs a fixed EndpointResolverV2 for deterministic endpoint routing.
func WithEndpointImmutable(url string) Option {
	return WithSrvOptionFuncs(
		func(o *s3.Options) {
			o.EndpointResolverV2 = &endpointResolver{url: url}
		},
	)
}

// endpointResolver resolves all S3 requests to a fixed endpoint URL.
type endpointResolver struct {
	url string
}

// ResolveEndpoint parses and returns the configured fixed endpoint URL.
func (r *endpointResolver) ResolveEndpoint(_ context.Context, _ s3.EndpointParameters) (
	sep.Endpoint,
	error,
) {
	u, err := url.Parse(r.url)
	if err != nil {
		return sep.Endpoint{}, err //nolint:wrapcheck
	}

	return sep.Endpoint{URI: *u}, nil
}
