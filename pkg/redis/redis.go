/*
Package redis provides a simple and basic wrapper client for interacting with
Redis (https://redis.io), an in-memory data store.

Based on https://github.com/redis/go-redis, it abstracts away the complexities
of the Redis protocol and provides a simplified interface.

This package includes functions for setting, getting, and deleting key/value
entries. Additionally, it supports sending and receiving messages from channels.

It allows to specify custom message encoding and decoding functions, including
serialization and encryption.
*/
package redis
