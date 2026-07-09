package s3

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/tecnickcom/gogen/pkg/awsopt"
)

// cfg stores AWS and service-specific settings used to construct an S3 client.
type cfg struct {
	awsConfig aws.Config
	awsOpts   awsopt.Options
	srvOptFns []SrvOptionFunc
	s3Client  S3
}

// loadConfig applies options and resolves aws.Config for S3 client construction.
//
// When a client is injected via [WithS3Client], the AWS configuration is neither
// loaded nor used: the injected client fully replaces the SDK client, so loading
// it would only add latency and a spurious failure mode (e.g. EC2 IMDS probing
// in an isolated/unit-test environment).
func loadConfig(ctx context.Context, opts ...Option) (*cfg, error) {
	c := &cfg{}

	for _, apply := range opts {
		apply(c)
	}

	if c.s3Client != nil {
		// The injected client replaces the SDK client, so awsConfig and
		// srvOptFns are never consumed: skip the (potentially slow or failing)
		// SDK config load entirely.
		return c, nil
	}

	awsConfig, err := c.awsOpts.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to load AWS configuration: %w", err)
	}

	c.awsConfig = awsConfig

	return c, nil
}
