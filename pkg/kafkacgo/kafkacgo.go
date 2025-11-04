/*
Package kafkacgo provides a simple high-level API for producing and consuming
Apache Kafka messages.

Based on github.com/confluentinc/confluent-kafka-go/kafka, it abstracts away the
complexities of the Kafka protocol and provides a simplified interface for
working with Kafka topics.

It allows to specify custom message encoding and decoding functions, including
serialization and encryption.

NOTE: This package depends on a C implementation, CGO must be enabled to use
this package. For a non-CGO implementation see the
github.com/Vonage/gosrvlib/pkg/kafka package.
*/
package kafkacgo
