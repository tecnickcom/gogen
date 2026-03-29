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

# What It Provides

  - [New] to construct a client from go-redis options.
  - Key/value operations: [Client.Set], [Client.Get], [Client.Del].
  - Pub/Sub operations: [Client.Send], [Client.Receive].
  - Typed data helpers: [Client.SetData], [Client.GetData], [Client.SendData],
    [Client.ReceiveData].
  - Connectivity probe: [Client.HealthCheck].
  - Clean shutdown of Pub/Sub and client resources via [Client.Close].

# Typed Payload Pipeline

By default, typed helpers use package [encode] for serialization. You can
replace this behavior with custom codec functions:

  - [WithMessageEncodeFunc]
  - [WithMessageDecodeFunc]

This makes it easy to introduce domain-specific serialization, compression, or
encryption without changing call sites.

# Subscription Configuration

Use options to define Pub/Sub behavior at client creation time:

  - [WithSubscrChannels] to subscribe to channels.
  - [WithSubscrChannelOptions] to tune subscription channel behavior.

# Benefits

  - Less Redis integration boilerplate in application code.
  - Consistent typed message handling across KV and Pub/Sub workflows.
  - Better testability through interface-based client abstractions.

# Usage

	srv := &redis.SrvOptions{Addr: "localhost:6379"}

	c, err := redis.New(ctx, srv, redis.WithSubscrChannels("events"))
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

	if err := c.HealthCheck(ctx); err != nil {
	    return err
	}
*/
package redis
