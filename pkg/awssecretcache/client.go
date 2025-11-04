package awssecretcache

import (
	"context"
	"fmt"
	"time"

	"github.com/Vonage/gosrvlib/pkg/sfcache"
	"github.com/aws/aws-sdk-go-v2/aws"
	awssm "github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

// Cache is a wrapper for the SecretsManager client in the AWS SDK.
type Cache struct {
	cache *sfcache.Cache
}

// New creates a new instance of the AWS SecretsManager cache.
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

// GetSecretData retrieves the data of the specified secret key (SecretId).
// Duplicate calls for the same key will wait for the first external call to complete (single-flight).
// It also handles the case where the cache entry is removed or updated during the wait.
// The function returns the cached value if available; otherwise, it performs a new external call.
// If the external call is successful, it updates the cache with the newly obtained value.
func (c *Cache) GetSecretData(ctx context.Context, key string) (*awssm.GetSecretValueOutput, error) {
	val, err := c.cache.Lookup(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve secret id %s: %w", key, err)
	}

	return val.(*awssm.GetSecretValueOutput), nil //nolint:forcetypeassert
}

// GetSecretBinary retrieves the decrypted binary value of the specified secret key (SecretId).
// If the secret is stored as a string, it will be converted to a byte slice.
// Uses: GetSecretData.
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

// GetSecretString retrieves the decrypted string value of the specified secret key (SecretId).
// If the secret is stored as a binary, it will be converted to a string.
// Uses: GetSecretData.
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

// Len returns the number of items in the cache.
func (c *Cache) Len() int {
	return c.cache.Len()
}

// Reset clears the whole cache.
func (c *Cache) Reset() {
	c.cache.Reset()
}

// Remove removes the cache entry for the specified key.
func (c *Cache) Remove(key string) {
	c.cache.Remove(key)
}
