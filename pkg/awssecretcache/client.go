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

// New constructs a Secrets Manager cache with single-flight lookups and TTL-based
// storage. The returned Cache is safe for concurrent use.
//
// A size <= 0 is clamped to a capacity of 1. A ttl <= 0 disables value caching (every
// call performs a fresh upstream lookup) while still coalescing concurrent lookups for
// the same key.
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

	return &Cache{
		cache: sfcache.New(lookupFn, sfcache.Config{
			Size:              size,
			TTL:               ttl,
			MaxStale:          cfg.maxStale,
			MaxStaleOnFailure: cfg.maxStaleOnFailure,
		}),
	}, nil
}

// GetSecretData returns the full Secrets Manager response for key.
//
// On cache hit, it serves data from memory. On cache miss or expiry, one
// goroutine performs the upstream GetSecretValue call while concurrent callers
// for the same key wait and share that result (single-flight behavior).
//
// Use it when callers need the metadata in GetSecretValueOutput in addition to the
// secret payload.
//
// Error fan-out: when the upstream GetSecretValue call fails, that error is
// shared with the callers coalesced into the same in-flight lookup, with two
// exceptions: a failure caused by the initiating caller's own context makes
// one waiting caller retry the lookup with its own context instead, and a
// caller whose own context ends while waiting receives an error wrapping
// [github.com/tecnickcom/nurago/pkg/sfcache.ErrLookupAborted]. The first
// exception is matched on the error's identity, so an SDK timeout error, which
// wraps context.DeadlineExceeded, also makes one waiting caller retry when the
// initiating caller's context has ended too: that costs one extra upstream call
// against an already timing-out endpoint, never a wrong result. Failed lookups
// are not cached, so a subsequent call after the flight completes triggers a
// fresh upstream request. Enable [WithStaleOnFailure] or [WithStaleIfError] to
// serve the last known good secret during upstream outages.
//
// With either stale option enabled, a failed refresh returns the last
// successfully fetched secret with a NIL error: callers cannot tell a stale
// secret from a freshly fetched one, and a secret rotated upstream during an
// outage keeps being served until the window closes.
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
// otherwise SecretString is returned directly. When the response holds no value at
// all (neither SecretString nor SecretBinary), it returns [ErrEmptySecret].
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
// It is not the cache's occupancy against its capacity: it also counts the secrets
// being fetched and the residue of a failed fetch, neither of which holds a value, so
// it can exceed the configured size. The overrun is bounded by the caller's own
// request concurrency plus two, never by the number of distinct secrets requested.
//
// During an outage, a stale serve that could evict nothing holds one secret more (see
// [WithStaleOnFailure]). It is reclaimed by the next secret successfully fetched.
func (c *Cache) Len() int {
	return c.cache.Len()
}

// Reset removes all cached entries.
//
// Fetches in flight are invalidated: their results are returned to the callers
// that started them but not cached, and callers waiting on them are released to
// fetch again.
func (c *Cache) Reset() {
	c.cache.Reset()
}

// Remove evicts key from the cache.
//
// A fetch in flight for key is invalidated: its result is returned to the caller
// that started it but not cached, and callers waiting on it are released to
// fetch again. Removing a secret whose fetch is still in flight therefore allows
// a second concurrent upstream call for it — the price of not serving the
// pre-rotation value.
func (c *Cache) Remove(key string) {
	c.cache.Remove(key)
}

// PurgeExpired removes all expired entries from the cache and returns the number of
// entries removed. It bounds how long expired secret material stays in process memory:
// expired entries are otherwise removed only when capacity pressure or a new lookup
// replaces them.
//
// NOTE: it also removes the values retained by [WithStaleIfError] and
// [WithStaleOnFailure], forfeiting stale protection for those keys until the next
// successful lookup.
func (c *Cache) PurgeExpired() int {
	return c.cache.PurgeExpired()
}
