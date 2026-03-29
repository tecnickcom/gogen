/*
Package awssecretcache solves the performance and reliability problem of
repeated AWS Secrets Manager lookups in high-throughput Go services. Fetching a
secret on every request adds network latency and can exhaust API quotas; this
package eliminates those costs with a local, in-memory, TTL-based cache backed
by a single-flight deduplication layer.

# Problem

AWS Secrets Manager calls are synchronous, network-bound, and subject to
throttling. Applications that resolve secrets per-request — for database
credentials, API keys, or feature flags — can easily become bottlenecked or
rate-limited. A naive local cache helps, but concurrent goroutines racing to
refresh an expired entry still trigger multiple redundant API calls. This
package fixes both problems at once.

# How It Works

[New] creates a [Cache] that wraps an aws-sdk-go-v2 SecretsManager client
(https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/secretsmanager) and
composes it with [github.com/tecnickcom/gogen/pkg/sfcache] — a fixed-size,
single-flight cache. The lookup flow is:

 1. On the first call for a given SecretId the cache is cold; one goroutine
    makes the real AWS API call while all other concurrent callers for the same
    key wait and share the result (single-flight).
 2. The result is stored in the fixed-size LRU cache with the configured TTL.
 3. Subsequent calls within the TTL window are served entirely from memory.
 4. After TTL expiry the entry is evicted and the next call triggers a fresh
    lookup with the same single-flight guarantees.

# Key Features

  - Single-flight deduplication: only one in-flight AWS API call per secret at
    any moment, regardless of goroutine concurrency. Prevents thundering-herd
    storms on TTL expiry.
  - TTL-based expiry: each entry lives for a caller-defined duration, ensuring
    secrets are refreshed regularly for rotation compliance.
  - Fixed-size cache: the maximum number of entries is set at construction time
    via the size parameter of [New], bounding memory usage predictably.
  - Thread-safe: all cache operations are safe for concurrent use with no
    external synchronization required.
  - Flexible secret retrieval: [Cache.GetSecretData] returns the raw SDK output;
    [Cache.GetSecretString] and [Cache.GetSecretBinary] transparently handle
    both storage formats (string and binary), respectively.
  - Manual cache control: [Cache.Remove] evicts a single entry (useful after a
    secret rotation event) and [Cache.Reset] clears the entire cache.
  - Pluggable AWS configuration: [WithAWSOptions], [WithSrvOptionFuncs],
    [WithEndpointMutable], and [WithEndpointImmutable] cover every AWS SDK
    customisation need, including local testing against mock endpoints.
  - Mockable client: [WithSecretsManagerClient] injects a custom
    [SecretsManagerClient], making unit tests fast and dependency-free.

# Usage

	cache, err := awssecretcache.New(
	    ctx,
	    128,              // maximum number of cached secrets
	    5*time.Minute,   // TTL per entry
	    awssecretcache.WithEndpointMutable("https://secretsmanager.us-east-1.amazonaws.com"),
	)
	if err != nil {
	    return err
	}

	value, err := cache.GetSecretString(ctx, "prod/myapp/db-password")
	if err != nil {
	    return err
	}

After a secret rotation, evict the stale entry immediately:

	cache.Remove("prod/myapp/db-password")

This package is ideal for any Go application — microservices, Lambda functions,
batch jobs — that relies heavily on AWS Secrets Manager and needs low-latency,
quota-friendly secret resolution.
*/
package awssecretcache
