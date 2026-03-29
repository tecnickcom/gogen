/*
Package sqs solves the operational friction of using AWS SQS directly from the
aws-sdk-go-v2 in application code. It wraps
github.com/aws/aws-sdk-go-v2/service/sqs with a small, focused API that covers
the common queue workflow: send, receive, decode, acknowledge (delete), and
health-check.

# Problem

Raw SQS integration requires repetitive boilerplate across services: queue and
endpoint setup, FIFO message-group handling, long-poll configuration,
visibility-timeout tuning, payload serialization, and error-safe delete logic.
Teams often re-implement this stack in each service, which leads to subtle
behavior drift and harder testing.

This package standardizes that flow into one client with explicit options and
safe defaults.

# How It Works

[New] creates a [Client] bound to a queue URL and optional message-group ID.

  - For FIFO queues (URL ends with `.fifo`), a valid message-group ID is
    required; for standard queues it must be omitted.
  - Long polling and visibility are configurable via [WithWaitTimeSeconds] and
    [WithVisibilityTimeout], with defaults of [DefaultWaitTimeSeconds] (20s)
    and [DefaultVisibilityTimeout] (600s).
  - Payloads can be sent/received as raw strings ([Client.Send],
    [Client.Receive]) or typed data ([Client.SendData], [Client.ReceiveData])
    through pluggable encode/decode hooks.

Receive flow behavior:

  - [Client.Receive] returns nil,nil when no message arrives within the
    long-poll window.
  - [Client.ReceiveData] returns an empty receipt handle when no message is
    available.
  - After successful processing, callers should acknowledge via
    [Client.Delete] using the receipt handle.
  - If decode fails, [Client.ReceiveData] still returns the receipt handle so
    callers can choose whether to delete or re-queue according to their policy.

# Key Features

  - Simple SQS workflow API: send, receive, delete, and health check.
  - FIFO-aware safety: validates message-group usage for FIFO queues.
  - Typed payload support with customizable serialization/encryption via
    [WithMessageEncodeFunc] and [WithMessageDecodeFunc].
  - Endpoint and AWS config customization via [WithAWSOptions],
    [WithSrvOptionFuncs], [WithEndpointMutable], and [WithEndpointImmutable]
    for local testing and advanced deployments.
  - Health probe support through [Client.HealthCheck], which verifies queue
    accessibility in the configured region.

# Usage

	c, err := sqs.New(ctx,
	    "https://sqs.us-east-1.amazonaws.com/123456789012/my-queue",
	    "", // non-FIFO queue
	    sqs.WithWaitTimeSeconds(20),
	    sqs.WithVisibilityTimeout(300),
	)
	if err != nil {
	    return err
	}

	// Send typed payload
	if err := c.SendData(ctx, event); err != nil {
	    return err
	}

	// Receive typed payload
	var msg Event
	receiptHandle, err := c.ReceiveData(ctx, &msg)
	if err != nil {
	    return err
	}
	if receiptHandle != "" {
	    _ = c.Delete(ctx, receiptHandle)
	}

This package is ideal for Go services that need a minimal, consistent,
production-friendly abstraction over SQS without losing access to AWS SDK
customization.
*/
package sqs
