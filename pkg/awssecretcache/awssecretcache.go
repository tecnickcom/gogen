/*
Package awssecretcache provides a simple client to retrieve and cache secrets from AWS Secrets Manager.
To improve speed and reduce costs, the client uses a thread-safe local single-flight cache
that avoids duplicate calls for the same secret.
The cache has a maximum size and a time-to-live (TTL) for each entry.
Duplicate calls for the same secret will wait for the first lookup to complete and return the same value.

This package is based on the official aws-sdk-go-v2 library.

Reference: https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/secretsmanager.
*/
package awssecretcache
