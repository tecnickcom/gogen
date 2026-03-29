/*
Package kafkacgo provides a high-level producer/consumer API for Apache Kafka
backed by Confluent's librdkafka Go bindings.

# Problem

The native Kafka protocol and client configuration surface are large and can be
verbose for common publish/consume workflows. Application code usually needs a
smaller API that still supports topic subscription, offset-reset policy,
session tuning, and message serialization hooks.

# Solution

This package wraps github.com/confluentinc/confluent-kafka-go/v2/kafka with a
minimal interface:

  - [NewProducer] + [Producer.Send] / [Producer.SendData]
  - [NewConsumer] + [Consumer.Receive] / [Consumer.ReceiveData]

It also exposes functional options for common Kafka client parameters and for
pluggable message codecs.

# Message Encoding and Decoding

By default, typed payloads are serialized/deserialized with
github.com/tecnickcom/gogen/pkg/encode:

  - [DefaultMessageEncodeFunc] for [Producer.SendData]
  - [DefaultMessageDecodeFunc] for [Consumer.ReceiveData]

You can replace both hooks via [WithMessageEncodeFunc] and
[WithMessageDecodeFunc] to introduce custom wire formats, compression,
authentication wrappers, or encryption.

# Features

  - High-level API over librdkafka for common produce/consume operations.
  - Configurable consumer behavior via options such as
    [WithAutoOffsetResetPolicy] and [WithSessionTimeout].
  - Configurable producer buffering via [WithProduceChannelSize].
  - Direct passthrough for arbitrary librdkafka configuration entries via
    [WithConfigParameter].
  - Typed message convenience methods ([Producer.SendData],
    [Consumer.ReceiveData]) with pluggable codecs.
  - Explicit lifecycle management with [Producer.Close] and [Consumer.Close].

# CGO Requirement

This package depends on a C implementation (librdkafka), therefore CGO must be
enabled to build and run it.

For a pure-Go alternative that does not require CGO, use:
  - github.com/tecnickcom/gogen/pkg/kafka

# Benefits

kafkacgo keeps Kafka integration concise while preserving access to advanced
broker/client tuning and custom payload pipelines, making it suitable for both
simple event publishing and production-grade streaming workloads.
*/
package kafkacgo
