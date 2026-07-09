package awssecretcache

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awssm "github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/tecnickcom/gogen/pkg/awsopt"
)

// SecretsManagerClient defines the AWS Secrets Manager calls used by this package.
type SecretsManagerClient interface {
	GetSecretValue(ctx context.Context, params *awssm.GetSecretValueInput, optFns ...func(*awssm.Options)) (*awssm.GetSecretValueOutput, error)
}

type cfg struct {
	// awsOpts holds the AWS SDK configuration options.
	awsOpts awsopt.Options

	// awsConfig holds the loaded AWS SDK configuration.
	awsConfig aws.Config

	// srvOptFns holds the options for the Secrets Manager client.
	srvOptFns []SrvOptionFunc

	// smclient is the AWS SDK Secrets Manager client.
	smclient SecretsManagerClient

	// maxStale bounds how long past its TTL a secret may be served when a
	// refresh fails (stale-if-error). Zero disables it.
	maxStale time.Duration
}

// loadConfig applies options and materializes the AWS SDK configuration.
//
// It centralizes option processing so New can build the cache from one
// validated cfg value. When no client is injected, it guarantees that
// awsConfig is loaded once with all collected awsopt options before any Secrets
// Manager client is used. When a client is injected via
// [WithSecretsManagerClient], the AWS configuration is neither loaded nor used:
// the injected client fully replaces the SDK client, so loading it would only
// add latency and a spurious failure mode (e.g. EC2 IMDS probing in an
// isolated/unit-test environment).
func loadConfig(ctx context.Context, opts ...Option) (*cfg, error) {
	c := &cfg{}

	for _, apply := range opts {
		apply(c)
	}

	if c.smclient != nil {
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
