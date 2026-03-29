/*
Package kafka provides a pure-Go, high-level API for producing and consuming
Apache Kafka messages.

# Problem

Kafka client libraries expose many low-level details (reader/writer setup,
group configuration, start offset behavior, serialization, lifecycle
management) that can make common publish/consume workflows verbose.
Applications often need a smaller API that is easy to adopt while still
allowing custom payload processing.

# Solution

Built on github.com/segmentio/kafka-go, this package wraps the core workflow
into a compact producer/consumer interface:

  - [NewProducer] + [Producer.Send] / [Producer.SendData]
  - [NewConsumer] + [Consumer.Receive] / [Consumer.ReceiveData]

Functional options provide targeted tuning (session timeout, start offset,
custom codecs) without requiring callers to manage the full kafka-go config
surface.

# Message Encoding and Decoding

Typed payload methods use configurable codec hooks:

  - [DefaultMessageEncodeFunc] powers [Producer.SendData]
  - [DefaultMessageDecodeFunc] powers [Consumer.ReceiveData]

Both defaults use github.com/tecnickcom/gogen/pkg/encode. Replace them via
[WithMessageEncodeFunc] and [WithMessageDecodeFunc] to add custom wire formats,
encryption, compression, or schema validation.

# Features

  - Pure-Go implementation (no CGO required).
  - Consumer group support with configurable session timeout via
    [WithSessionTimeout].
  - Configurable starting point for unread offsets via [WithFirstOffset]
    (default is latest offset).
  - Blocking receive semantics through [Consumer.Receive], suitable for worker
    loops driven by context cancellation.
  - Health probe support through [Consumer.HealthCheck] to verify broker/topic
    reachability from the process.
  - Explicit resource lifecycle with [Producer.Close] and [Consumer.Close].

# Implementation Choice

This package is the non-CGO Kafka client for gogen.
If you need librdkafka/Confluent-specific behavior, see:
  - github.com/tecnickcom/gogen/pkg/kafkacgo

# Benefits

The package offers a pragmatic balance between simplicity and extensibility,
helping teams adopt Kafka quickly with a clean API while retaining control over
encoding, offsets, and runtime tuning.
*/
package kafka
