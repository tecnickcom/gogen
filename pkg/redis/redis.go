/*
Package redis provides focused helpers built on go-redis for common application
workflows: key/value storage, Pub/Sub messaging, typed payload encoding, and
basic connection health checks.

Redis reference: https://redis.io
Underlying client: https://github.com/redis/go-redis

# Problem

Direct use of go-redis is powerful but often leads to repeated adapter code in
services: typed payload serialization, subscription channel wiring, and
consistent error wrapping around set/get/publish operations. This package
centralizes those patterns behind a minimal client API.

# How It Works

[New] creates a [Client] given a [SrvOptions] (aliased from go-redis Options)
and a variadic list of [Option] values:

 1. The server address is validated before any network connection is attempted
    (skipped when a client is injected via [WithRedisClient], since no
    connection is dialed). Both TCP host:port addresses and unix domain
    sockets are supported.
 2. A go-redis client is constructed (or injected via [WithRedisClient] for
    tests). When at least one channel is declared with [WithChannels], a
    Pub/Sub subscription feeds [Client.Receive] and [Client.ReceiveData] one
    message per call. The subscription runs until [Client.Close] is called:
    canceling the [New] context does not stop it.
 3. Encode and decode functions — defaulting to [DefaultMessageEncodeFunc] and
    [DefaultMessageDecodeFunc] — are stored on the client and used
    transparently by the typed data methods.

# Key Features

  - Raw operations: [Client.Set], [Client.Get], and [Client.Del] offer direct
    key/value access with expiration support.
  - Typed data operations: [Client.SetData] and [Client.GetData] encode/decode
    Go values automatically using the configured [TEncodeFunc] / [TDecodeFunc],
    removing serialization boilerplate at every call site.
  - Pub/Sub: [Client.Send] and [Client.Receive] work with raw strings;
    [Client.SendData] and [Client.ReceiveData] apply the same encode/decode
    pipeline, returning the channel name alongside the decoded value.
  - Pluggable serialization: [WithMessageEncodeFunc] and
    [WithMessageDecodeFunc] replace the default codec with any implementation —
    including encrypted or compressed payloads — without changing call sites.
  - Health check: [Client.HealthCheck] sends a PING and returns a wrapped error
    on failure, making it trivial to integrate with liveness probes.
  - Sentinel errors: a missing key surfaces as [ErrKeyNotFound], and
    configuration or subscription states as the other exported Err values,
    all matchable with errors.Is.
  - Mockable client: [WithRedisClient] injects a custom [RClient], enabling
    fast, dependency-free unit tests of the command path.
  - Graceful close: [Client.Close] releases Pub/Sub and client resources; it is
    idempotent and required to stop the subscription when channels are
    configured.

# Subscription Configuration

Use options to define Pub/Sub behavior at client creation time:

  - [WithChannels] to subscribe to channels.
  - [WithChannelOptions] to tune subscription channel behavior: buffer size,
    send timeout, and health check interval. Note that with the go-redis
    defaults, a consumer that stops calling [Client.Receive] loses messages
    once the 100-message buffer stays full for one minute.

# Usage

	srv := &redis.SrvOptions{Addr: "localhost:6379"}

	c, err := redis.New(ctx, srv, redis.WithChannels("events"))
	if err != nil {
	    return err
	}
	defer c.Close()

	if err := c.Set(ctx, "k", "v", 0); err != nil {
	    return err
	}

	if err := c.SendData(ctx, "events", event); err != nil {
	    return err
	}

	var event Event

	channel, err := c.ReceiveData(ctx, &event)
	if err != nil {
	    return err
	}

	if err := c.HealthCheck(ctx); err != nil {
	    return err
	}

To swap in an encrypted codec, supply custom functions at construction time:

	c, err := redis.New(ctx, srv,
	    redis.WithMessageEncodeFunc(myEncryptAndEncode),
	    redis.WithMessageDecodeFunc(myDecryptAndDecode),
	)
*/
package redis
