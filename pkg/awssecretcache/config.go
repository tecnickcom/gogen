package awssecretcache

import (
	"context"
	"fmt"

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
}

// loadConfig applies options and materializes the AWS SDK configuration.
//
// It centralizes option processing so New can build the cache from one
// validated cfg value. The function guarantees that awsConfig is loaded once
// with all collected awsopt options before any Secrets Manager client is used.
func loadConfig(ctx context.Context, opts ...Option) (*cfg, error) {
	c := &cfg{}

	for _, apply := range opts {
		apply(c)
	}

	awsConfig, err := c.awsOpts.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to load AWS configuration: %w", err)
	}

	c.awsConfig = awsConfig

	return c, nil
}
