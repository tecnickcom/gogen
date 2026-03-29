package awssecretcache

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awssm "github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/tecnickcom/gogen/pkg/sfcache"
)

// Cache provides TTL and single-flight caching for AWS Secrets Manager lookups.
type Cache struct {
	cache *sfcache.Cache[string]
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

	lookupFn := func(ctx context.Context, key string) (any, error) {
		input := &awssm.GetSecretValueInput{
			SecretId: aws.String(key),
		}

		return smclient.GetSecretValue(ctx, input)
	}

	return &Cache{
		cache: sfcache.New(lookupFn, size, ttl),
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
func (c *Cache) GetSecretData(ctx context.Context, key string) (*awssm.GetSecretValueOutput, error) {
	val, err := c.cache.Lookup(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve secret id %s: %w", key, err)
	}

	return val.(*awssm.GetSecretValueOutput), nil //nolint:forcetypeassert
}

// GetSecretBinary returns key as bytes, regardless of how it is stored in AWS.
//
// If the secret is stored as SecretString, the value is converted to []byte;
// otherwise SecretBinary is returned directly. This gives callers a uniform
// binary-oriented API for downstream decoding or decryption workflows.
func (c *Cache) GetSecretBinary(ctx context.Context, key string) ([]byte, error) {
	val, err := c.GetSecretData(ctx, key)
	if err != nil {
		return nil, err
	}

	if val.SecretString != nil {
		return []byte(aws.ToString(val.SecretString)), nil
	}

	return val.SecretBinary, nil
}

// GetSecretString returns key as text, regardless of how it is stored in AWS.
//
// If the secret is stored as SecretBinary, the bytes are converted to string;
// otherwise SecretString is returned directly. This simplifies application code
// that expects textual secrets such as DSNs, API keys, or tokens.
func (c *Cache) GetSecretString(ctx context.Context, key string) (string, error) {
	val, err := c.GetSecretData(ctx, key)
	if err != nil {
		return "", err
	}

	if val.SecretString != nil {
		return aws.ToString(val.SecretString), nil
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
