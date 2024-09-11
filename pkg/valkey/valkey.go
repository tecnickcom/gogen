/*
Package valkey provides a simple and basic wrapper client for interacting with
Valkey (https://valkey.io), an open source in-memory data store.

Based on https://github.com/valkey-io/valkey-go, it abstracts away the complexities
of the valkey protocol and provides a simplified interface.

This package includes functions for setting, getting, and deleting key/value
entries. Additionally, it supports sending and receiving messages from channels.

It allows to specify custom message encoding and decoding functions, including
serialization and encryption.
*/
package valkey
