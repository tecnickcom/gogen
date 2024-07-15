/*
Package awssecretcache provides a simple client for retrieving and caching
secrets from AWS Secrets Manager.

This package is based on the official aws-sdk-go-v2 library
(https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/secretsmanager) and
implements github.com/Vonage/gosrvlib/pkg/sfcache to provide a simple, local,
thread-safe, fixed-size, and single-flight cache for AWS Secrets lookup calls.

By caching previous values, awssecretcache improves the performance of secrets
lookup by eliminating the need for repeated expensive requests.

This package provides a local in-memory cache with a configurable maximum number
of entries. The fixed size helps with efficient memory management and prevents
excessive memory usage. The cache is thread-safe, allowing concurrent access
without the need for external synchronization. It efficiently handles concurrent
requests by sharing results from the first lookup, ensuring that only one
request makes the expensive call, and avoiding unnecessary network load or
resource starvation. Duplicate calls for the same key will wait for the first
call to complete and return the same value.

Each cache entry has a set time-to-live (TTL), so it will automatically expire.
However, it is also possible to force the removal of a specific entry or reset
the entire cache.

This package is ideal for any Go application that heavily relies on AWS Secrets
lookups.
*/
package awssecretcache
