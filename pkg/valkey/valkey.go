/*
Package valkey solves the friction of integrating Valkey (https://valkey.io) —
the open-source, Redis-compatible in-memory data store — into a Go service.
The upstream valkey-go client (https://github.com/valkey-io/valkey-go) is
powerful but low-level; this package wraps it with a clean, opinionated API
that covers the most common patterns: key/value storage, typed data
serialization, and Pub/Sub messaging, all behind a single [Client] type.

# Problem

Working directly with valkey-go requires composing command builders,
type-asserting results, handling Pub/Sub subscriptions manually, and wiring in
serialization for every call site. Teams end up duplicating the same thin
adapter in every service. This package provides that adapter once, with
consistent error wrapping, pluggable encode/decode functions, and a
mockable interface for testing.

# How It Works

[New] creates a [Client] given a [SrvOptions] (aliased from
valkey-go's ClientOption) and a variadic list of [Option] values:

 1. The server address is validated before any network connection is attempted.
 2. A valkey-go client is constructed (or injected via [WithValkeyClient] for
    tests) and stored alongside a pre-built, pinned Pub/Sub subscription
    command covering all channels declared with [WithChannels].
 3. Encode and decode functions — defaulting to [DefaultMessageEncodeFunc] and
    [DefaultMessageDecodeFunc] — are stored on the client and used
    transparently by the typed data methods.

# Key Features

  - Raw string operations: [Client.Set], [Client.Get], and [Client.Del]
    offer direct key/value access with expiration support.
  - Typed data operations: [Client.SetData] and [Client.GetData] encode/decode
    Go values automatically using the configured [TEncodeFunc] / [TDecodeFunc],
    removing serialization boilerplate at every call site.
  - Pub/Sub: [Client.Send] and [Client.Receive] work with raw strings;
    [Client.SendData] and [Client.ReceiveData] apply the same encode/decode
    pipeline, returning the channel name alongside the decoded value.
  - Pluggable serialization: [WithMessageEncodeFunc] and
    [WithMessageDecodeFunc] replace the default JSON+base64 codec with any
    implementation — including encrypted or compressed payloads — without
    changing call sites.
  - Health check: [Client.HealthCheck] sends a PING and returns a wrapped error
    on failure, making it trivial to integrate with liveness probes.
  - Mockable client: [WithValkeyClient] injects a custom [VKClient], enabling
    fast, dependency-free unit tests.
  - Graceful close: [Client.Close] drains all pending calls before releasing
    the underlying connection.

# Usage

	srvOpts := valkey.SrvOptions{InitAddress: []string{"localhost:6379"}}

	client, err := valkey.New(
	    ctx,
	    srvOpts,
	    valkey.WithChannels("events", "notifications"),
	)
	if err != nil {
	    return err
	}
	defer client.Close()

	// Store and retrieve a typed value:
	type Payload struct{ Message string }

	if err := client.SetData(ctx, "my-key", Payload{"hello"}, time.Hour); err != nil {
	    return err
	}

	var p Payload
	if err := client.GetData(ctx, "my-key", &p); err != nil {
	    return err
	}

	// Publish and consume a typed message:
	if err := client.SendData(ctx, "events", Payload{"fired"}); err != nil {
	    return err
	}

	var event Payload
	channel, err := client.ReceiveData(ctx, &event)

To swap in an encrypted codec, supply custom functions at construction time:

	client, err := valkey.New(ctx, srvOpts,
	    valkey.WithMessageEncodeFunc(myEncryptAndEncode),
	    valkey.WithMessageDecodeFunc(myDecryptAndDecode),
	)
*/
package valkey
