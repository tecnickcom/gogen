/*
Package sqs provides a simple and basic wrapper client for interacting with AWS
SQS (Amazon Simple Queue Service).

Based on github.com/aws/aws-sdk-go-v2/service/sqs, it abstracts away the
complexities of the SQS protocol and provides a simplified interface.

This package includes functions for sending, receiving, and deleting messages.

It allows to specify custom message encoding and decoding functions, including
serialization and encryption.
*/
package sqs
