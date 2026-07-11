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
  - [NewConsumer] + [Consumer.FetchMessage] / [Consumer.CommitMessages]

Functional options provide targeted tuning (session timeout, start offset,
required acks, batching, custom codecs) without requiring callers to manage
the full kafka-go config surface.

# Delivery Semantics

Producer writes wait for broker acknowledgment from the full in-sync replica
set by default (kafka.RequireAll); tune this with [WithRequiredAcks].

On the consumer side, when a consumer group is configured:

  - [Consumer.Receive] and [Consumer.ReceiveData] are at-most-once: the offset
    is committed as soon as the message is read, before the caller processes
    it, so a crash or decode failure after the read permanently skips the
    message.
  - [Consumer.FetchMessage] + [Consumer.CommitMessages] are at-least-once: the
    offset is committed only when the caller explicitly acknowledges the
    message after successful processing.

When no consumer group is configured (empty groupID), offsets are never
committed: [Consumer.Receive] and [Consumer.FetchMessage] behave identically,
reading always starts from the earliest available offset, and
[Consumer.CommitMessages] returns an error.

# Message Encoding and Decoding

Typed payload methods use configurable codec hooks:

  - [DefaultMessageEncodeFunc] powers [Producer.SendData]
  - [DefaultMessageDecodeFunc] powers [Consumer.ReceiveData]

Both defaults use github.com/tecnickcom/nurago/pkg/encode. Replace them via
[WithMessageEncodeFunc] and [WithMessageDecodeFunc] to add custom wire formats,
encryption, compression, or schema validation.

# Errors

Configuration problems are reported at construction time with errors matching
the exported sentinels [ErrInvalidOptions], [ErrNilEncodeFunc], and
[ErrNilDecodeFunc]. After [Consumer.Close], the receive methods return errors
matching [ErrConsumerClosed]. Match the sentinels with errors.Is.

# Features

  - Pure-Go implementation (no CGO required).
  - Consumer group support with configurable session timeout via
    [WithSessionTimeout].
  - Configurable consumer-group start offset via [WithFirstOffset]: a group
    without a committed offset starts from the latest offset by default.
    Without a consumer group the option has no effect and reading always
    starts from the earliest available offset.
  - Blocking receive semantics through [Consumer.Receive], suitable for worker
    loops driven by context cancellation.
  - Explicit offset acknowledgment via [Consumer.FetchMessage] and
    [Consumer.CommitMessages] for at-least-once processing.
  - Configurable broker acknowledgment via [WithRequiredAcks] and producer
    batching via [WithBatchSize] / [WithBatchTimeout].
  - Health probe support through [Consumer.HealthCheck] and
    [Producer.HealthCheck] to verify broker/topic reachability from the
    process.
  - Mockable clients: [WithKafkaReader] and [WithKafkaWriter] inject custom
    [KReader] / [KWriter] implementations, enabling fast, dependency-free
    unit tests.
  - Explicit resource lifecycle with [Producer.Close] and [Consumer.Close].

# Implementation Choice

This package is a pure-Go Kafka client: it requires no CGO and no system
librdkafka installation, which keeps builds, cross-compilation, and minimal
container images (scratch/distroless) simple.

# Benefits

The package offers a pragmatic balance between simplicity and extensibility,
helping teams adopt Kafka quickly with a clean API while retaining control over
encoding, offsets, and runtime tuning.
*/
package kafka
