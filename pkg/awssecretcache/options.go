package awssecretcache

import (
	"context"
	"net/url"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	sep "github.com/aws/smithy-go/endpoints"
	"github.com/tecnickcom/nurago/pkg/awsopt"
)

// SrvOptionFunc is an alias for this service option function.
type SrvOptionFunc = func(*secretsmanager.Options)

// Option is a type to allow setting custom client options.
type Option func(*cfg)

// WithAWSOptions appends raw AWS SDK load options used to build aws.Config: region,
// credentials, retryers, and similar settings.
func WithAWSOptions(opt awsopt.Options) Option {
	return func(c *cfg) {
		c.awsOpts = append(c.awsOpts, opt...)
	}
}

// WithSrvOptionFuncs appends Secrets Manager service-level option functions, applied
// when constructing the underlying secretsmanager.Client.
func WithSrvOptionFuncs(opt ...SrvOptionFunc) Option {
	return func(c *cfg) {
		c.srvOptFns = append(c.srvOptFns, opt...)
	}
}

// WithSecretsManagerClient injects a custom SecretsManagerClient implementation, so no
// real secretsmanager.Client is built from aws.Config.
//
// When set, the injected client is used as-is: the AWS configuration is not
// loaded and the AWS/service options ([WithAWSOptions], [WithSrvOptionFuncs],
// [WithEndpointMutable], [WithEndpointImmutable]) are ignored.
func WithSecretsManagerClient(smclient SecretsManagerClient) Option {
	return func(c *cfg) {
		c.smclient = smclient
	}
}

// WithStaleIfError serves the last known good secret when a refresh fails,
// with the stale window anchored to the secret's original expiration
// (RFC 5861 stale-if-error, see [github.com/tecnickcom/nurago/pkg/sfcache.Config.MaxStale]).
//
// If fetching an expired secret returns an error (throttling, outage,
// timeout), the previously cached value is returned with a nil error
// instead, but only until its original expiration plus maxStale. Every call
// still attempts a fresh upstream lookup, so recovery is automatic on the
// first success. Callers cannot distinguish a stale secret from a fresh one,
// and stale protection is best-effort: the retained value is lost to cache
// eviction under capacity pressure, [Cache.PurgeExpired], [Cache.Remove],
// and [Cache.Reset]. A maxStale <= 0 disables the behavior (default).
//
// PRECONDITION: because the window is anchored to the expiration and not to the
// failure, only a secret read more often than ttl + maxStale is protected. One idle for
// longer than that has no stale protection at all: the outage error is returned even
// though the last known good value is still cached. Use [WithStaleOnFailure] to protect
// rarely read secrets.
//
// Size maxStale against the rotation policy, so a rotated-out secret cannot be served
// long after the rotation event.
func WithStaleIfError(maxStale time.Duration) Option {
	return func(c *cfg) {
		c.maxStale = maxStale
	}
}

// WithStaleOnFailure serves the last known good secret for up to
// maxStaleOnFailure after a refresh first fails, however long the secret had
// been idle before the failure (see
// [github.com/tecnickcom/nurago/pkg/sfcache.Config.MaxStaleOnFailure]).
//
// Unlike [WithStaleIfError] it holds for rarely read secrets. The window is anchored
// once, by the first failed refresh: further failures keep serving the same secret until
// that deadline but never push it back, so an outage cannot make a secret immortal.
// Every call still attempts a fresh upstream lookup and recovery is automatic on the
// first success. The caveats of [WithStaleIfError] otherwise apply unchanged: callers
// cannot distinguish a stale secret from a fresh one, and retention is best-effort.
//
// A maxStaleOnFailure <= 0 disables the behavior (default). Size it against the rotation
// policy, so a rotated-out secret cannot be served long after the rotation event. When
// both options are set, the secret is served stale until the later of the two deadlines.
func WithStaleOnFailure(maxStaleOnFailure time.Duration) Option {
	return func(c *cfg) {
		c.maxStaleOnFailure = maxStaleOnFailure
	}
}

// WithEndpointMutable sets BaseEndpoint on the SDK client. BaseEndpoint remains
// mutable, so the SDK can still adjust request routing details.
func WithEndpointMutable(url string) Option {
	return WithSrvOptionFuncs(
		func(o *secretsmanager.Options) {
			o.BaseEndpoint = aws.String(url)
		},
	)
}

// WithEndpointImmutable installs a custom EndpointResolverV2 with a fixed URL, so
// endpoint selection is deterministic and the default resolver logic is bypassed.
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
