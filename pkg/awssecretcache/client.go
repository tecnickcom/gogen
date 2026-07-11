package awssecretcache

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awssm "github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/tecnickcom/nurago/pkg/sfcache"
)

// Cache provides TTL and single-flight caching for AWS Secrets Manager lookups.
type Cache struct {
	cache *sfcache.Cache[string, *awssm.GetSecretValueOutput]
}

// New constructs a Secrets Manager cache with single-flight lookups and TTL-based storage.
//
// It addresses two common production problems: repeated network latency from
// fetching the same secret on every call, and duplicate upstream requests when
// many goroutines request an expired key at the same time.
//
// Key features:
//   - fixed-size cache capacity via size to bound memory use;
//   - TTL-driven refresh via ttl to keep rotated secrets up to date;
//   - option-based AWS and client customization for real or mocked backends.
//
// Edge cases: a size <= 0 is clamped to a capacity of 1, and a ttl <= 0
// disables value caching (every call performs a fresh upstream lookup) while
// still coalescing concurrent lookups for the same key via single-flight.
//
// The returned Cache is safe for concurrent use.
func New(ctx context.Context, size int, ttl time.Duration, opts ...Option) (*Cache, error) {
	cfg, err := loadConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("cannot create a new AWS secretsmanager client: %w", err)
	}

	smclient := cfg.smclient
	if smclient == nil {
		smclient = awssm.NewFromConfig(cfg.awsConfig, cfg.srvOptFns...)
	}

	lookupFn := func(ctx context.Context, key string) (*awssm.GetSecretValueOutput, error) {
		input := &awssm.GetSecretValueInput{
			SecretId: aws.String(key),
		}

		return smclient.GetSecretValue(ctx, input)
	}

	var sfopts []sfcache.Option[string, *awssm.GetSecretValueOutput]

	if cfg.maxStale > 0 {
		sfopts = append(sfopts, sfcache.WithStaleIfError[string, *awssm.GetSecretValueOutput](cfg.maxStale))
	}

	return &Cache{
		cache: sfcache.New(lookupFn, size, ttl, sfopts...),
	}, nil
}

// GetSecretData returns the full Secrets Manager response for key.
//
// On cache hit, it serves data from memory. On cache miss or expiry, one
// goroutine performs the upstream GetSecretValue call while concurrent callers
// for the same key wait and share that result (single-flight behavior).
//
// This reduces latency variance, avoids API bursts, and provides a single
// entry point when callers need metadata in GetSecretValueOutput in addition to
// the secret payload.
//
// Error fan-out: when the upstream GetSecretValue call fails, that error is
// shared with the callers coalesced into the same in-flight lookup, with two
// exceptions: a failure caused by the initiating caller's own context makes
// one waiting caller retry the lookup with its own context instead, and a
// caller whose own context ends while waiting receives an error wrapping
// [github.com/tecnickcom/nurago/pkg/sfcache.ErrLookupAborted]. Failed lookups
// are not cached, so a subsequent call after the flight completes triggers a
// fresh upstream request. Callers that need resilience against transient
// failures can enable [WithStaleIfError] to serve the last known good secret
// during upstream outages, or wrap this method in their own retry/backoff
// logic.
//
// The returned output is shared by reference with every other caller of the
// same key: treat it as read-only. Use [Cache.GetSecretBinary] or
// [Cache.GetSecretString] for values that are safe to modify.
//
// An empty key yields [ErrEmptySecretID] before any upstream call is made.
//
// Caching note: a value-less or nil response is cached like any successful
// value for the TTL, so [ErrEmptySecret] persists until expiry or an explicit
// [Cache.Remove]/[Cache.Reset]; genuine upstream errors are never cached and are
// retried on the next call.
//
// Error matching: a nil upstream response yields [ErrEmptySecret]. Underlying
// errors propagate through the %w wrapping, so callers can still match AWS SDK
// typed errors with errors.As (e.g. new(*types.ResourceNotFoundException) for a
// missing secret) and context/abort errors with errors.Is (against
// [github.com/tecnickcom/nurago/pkg/sfcache.ErrLookupAborted], [context.Canceled],
// or [context.DeadlineExceeded]).
func (c *Cache) GetSecretData(ctx context.Context, key string) (*awssm.GetSecretValueOutput, error) {
	if key == "" {
		// An empty secret id is always rejected by AWS: fail fast without a
		// wasted upstream call or a cache entry under the empty key.
		return nil, ErrEmptySecretID
	}

	val, err := c.cache.Lookup(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve secret id %s: %w", key, err)
	}

	if val == nil {
		// The AWS SDK never returns a nil output on success, but an injected
		// SecretsManagerClient can: guard against it so the getters never
		// dereference a nil output.
		return nil, fmt.Errorf("%w: %s", ErrEmptySecret, key)
	}

	return val, nil
}

// GetSecretBinary returns key as bytes, regardless of how it is stored in AWS.
//
// If the secret is stored as SecretString, the value is converted to []byte;
// otherwise a copy of SecretBinary is returned. The returned slice is never
// shared with the cache, so callers may safely zero it after use without
// corrupting the value served to other callers. When the response holds no
// value at all (neither SecretString nor SecretBinary), it returns
// [ErrEmptySecret].
func (c *Cache) GetSecretBinary(ctx context.Context, key string) ([]byte, error) {
	val, err := c.GetSecretData(ctx, key)
	if err != nil {
		return nil, err
	}

	if val.SecretString != nil {
		return []byte(aws.ToString(val.SecretString)), nil
	}

	if val.SecretBinary == nil {
		return nil, fmt.Errorf("%w: %s", ErrEmptySecret, key)
	}

	// Return a copy: the cached output is shared by reference across callers.
	return slices.Clone(val.SecretBinary), nil
}

// GetSecretString returns key as text, regardless of how it is stored in AWS.
//
// If the secret is stored as SecretBinary, the bytes are converted to string;
// otherwise SecretString is returned directly. This simplifies application code
// that expects textual secrets such as DSNs, API keys, or tokens. When the
// response holds no value at all (neither SecretString nor SecretBinary), it
// returns [ErrEmptySecret].
func (c *Cache) GetSecretString(ctx context.Context, key string) (string, error) {
	val, err := c.GetSecretData(ctx, key)
	if err != nil {
		return "", err
	}

	if val.SecretString != nil {
		return aws.ToString(val.SecretString), nil
	}

	if val.SecretBinary == nil {
		return "", fmt.Errorf("%w: %s", ErrEmptySecret, key)
	}

	return string(val.SecretBinary), nil
}

// Len reports the current number of cached entries.
//
// It is useful for observability and capacity tuning when choosing cache size
// and TTL values for workload patterns.
func (c *Cache) Len() int {
	return c.cache.Len()
}

// Reset removes all cached entries.
//
// Use it after broad secret rotation events or test setup/teardown when a
// full cache invalidation is preferred over key-by-key removal.
func (c *Cache) Reset() {
	c.cache.Reset()
}

// Remove evicts key from the cache.
//
// This allows targeted invalidation after rotating a single secret without
// disrupting other hot entries.
func (c *Cache) Remove(key string) {
	c.cache.Remove(key)
}

// PurgeExpired removes all expired entries from the cache and returns the
// number of entries removed.
//
// Use it (e.g. on a timer) to bound how long expired secret material stays
// in process memory: expired entries are otherwise only removed lazily,
// when capacity pressure or a new lookup replaces them. Note that it also
// removes values retained by [WithStaleIfError], forfeiting stale
// protection for those keys until the next successful lookup.
func (c *Cache) PurgeExpired() int {
	return c.cache.PurgeExpired()
}
