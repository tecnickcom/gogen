package sqs

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/tecnickcom/nurago/pkg/awsopt"
)

const (
	// DefaultWaitTimeSeconds is the default duration (in seconds) for which the call waits for a message to arrive in the queue before returning.
	// This must be between 0 and 20 seconds.
	DefaultWaitTimeSeconds = 20

	// DefaultVisibilityTimeout is the default duration (in seconds) that the received messages are hidden from subsequent retrieve requests after being retrieved by a ReceiveMessage request.
	DefaultVisibilityTimeout = 600

	// maxWaitTimeSeconds is the maximum long-poll wait supported by SQS ReceiveMessage.
	maxWaitTimeSeconds = 20

	// maxVisibilityTimeout is the maximum visibility timeout supported by SQS (12 hours).
	maxVisibilityTimeout = 43200
)

// cfg stores validated SQS runtime configuration and codec hooks.
type cfg struct {
	awsOpts           awsopt.Options
	awsConfig         aws.Config
	srvOptFns         []SrvOptionFunc
	sqsClient         SQS
	waitTimeSeconds   int32
	visibilityTimeout int32
	messageEncodeFunc TEncodeFunc
	messageDecodeFunc TDecodeFunc
}

// loadConfig applies options, validates boundaries/codecs, and resolves aws.Config.
//
// queueURL is used only to derive a fallback AWS region (see below). When a
// client is injected via [WithSQSClient], the AWS configuration is neither
// loaded nor used: the injected client fully replaces the SDK client, so loading
// it would only add latency and a spurious failure mode (e.g. EC2 IMDS probing
// in an isolated/unit-test environment).
func loadConfig(ctx context.Context, queueURL string, opts ...Option) (*cfg, error) {
	c := &cfg{
		waitTimeSeconds:   DefaultWaitTimeSeconds,
		visibilityTimeout: DefaultVisibilityTimeout,
		messageEncodeFunc: DefaultMessageEncodeFunc,
		messageDecodeFunc: DefaultMessageDecodeFunc,
	}

	for _, apply := range opts {
		apply(c)
	}

	if c.messageEncodeFunc == nil {
		return nil, ErrNilEncodeFunc
	}

	if c.messageDecodeFunc == nil {
		return nil, ErrNilDecodeFunc
	}

	if c.waitTimeSeconds < 0 || c.waitTimeSeconds > maxWaitTimeSeconds {
		return nil, ErrInvalidWaitTime
	}

	if c.visibilityTimeout < 0 || c.visibilityTimeout > maxVisibilityTimeout {
		return nil, ErrInvalidVisibilityTimeout
	}

	if c.sqsClient != nil {
		// The injected client replaces the SDK client, so awsConfig and
		// srvOptFns are never consumed: skip the (potentially slow or failing)
		// SDK config load entirely.
		return c, nil
	}

	// Derive the region from the queue URL as a fallback so callers need not
	// repeat it; an explicit region from WithAWSOptions (appended later, so it
	// wins) still overrides it, and a URL without a region-shaped label adds
	// nothing.
	awsOpts := make(awsopt.Options, 0, len(c.awsOpts)+1)
	awsOpts.WithRegionFromURL(queueURL, "")
	awsOpts = append(awsOpts, c.awsOpts...)

	awsConfig, err := awsOpts.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to load AWS configuration: %w", err)
	}

	c.awsConfig = awsConfig

	return c, nil
}
