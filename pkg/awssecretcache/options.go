package awssecretcache

import (
	"context"
	"net/url"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	sep "github.com/aws/smithy-go/endpoints"
	"github.com/tecnickcom/gogen/pkg/awsopt"
)

// SrvOptionFunc is an alias for this service option function.
type SrvOptionFunc = func(*secretsmanager.Options)

// Option is a type to allow setting custom client options.
type Option func(*cfg)

// WithAWSOptions appends raw AWS SDK load options used to build aws.Config.
//
// Use this to pass shared awsopt.Options (region, credentials, retryers, and
// similar settings) so the cache client follows the same AWS behavior as other
// services in the process.
func WithAWSOptions(opt awsopt.Options) Option {
	return func(c *cfg) {
		c.awsOpts = append(c.awsOpts, opt...)
	}
}

// WithSrvOptionFuncs appends Secrets Manager service-level option functions.
//
// These options are applied when constructing the underlying
// secretsmanager.Client, enabling per-service customization beyond global
// aws.Config settings.
func WithSrvOptionFuncs(opt ...SrvOptionFunc) Option {
	return func(c *cfg) {
		c.srvOptFns = append(c.srvOptFns, opt...)
	}
}

// WithSecretsManagerClient injects a custom SecretsManagerClient implementation.
//
// This is primarily useful for tests and advanced integrations where the
// caller needs full control over request behavior without creating a real
// secretsmanager.Client from aws.Config.
func WithSecretsManagerClient(smclient SecretsManagerClient) Option {
	return func(c *cfg) {
		c.smclient = smclient
	}
}

// WithEndpointMutable sets BaseEndpoint on the SDK client.
//
// Because BaseEndpoint remains mutable, the SDK can still adjust request
// routing details. This is a practical choice for local stacks and some proxy
// setups where endpoint customization should not fully replace SDK resolution.
func WithEndpointMutable(url string) Option {
	return WithSrvOptionFuncs(
		func(o *secretsmanager.Options) {
			o.BaseEndpoint = aws.String(url)
		},
	)
}

// WithEndpointImmutable installs a custom EndpointResolverV2 with a fixed URL.
//
// This makes endpoint selection deterministic and bypasses default resolver
// logic, which is useful when all requests must target an exact endpoint.
func WithEndpointImmutable(url string) Option {
	return WithSrvOptionFuncs(
		func(o *secretsmanager.Options) {
			o.EndpointResolverV2 = &endpointResolver{url: url}
		},
	)
}

// endpointResolver is a custom endpoint resolver.
type endpointResolver struct {
	url string
}

// ResolveEndpoint parses and returns the fixed endpoint configured on r.
//
// It is used by WithEndpointImmutable to provide a stable EndpointResolverV2
// implementation for the Secrets Manager client.
func (r *endpointResolver) ResolveEndpoint(_ context.Context, _ secretsmanager.EndpointParameters) (
	sep.Endpoint,
	error,
) {
	u, err := url.Parse(r.url)
	if err != nil {
		return sep.Endpoint{}, err //nolint:wrapcheck
	}

	return sep.Endpoint{URI: *u}, nil
}
