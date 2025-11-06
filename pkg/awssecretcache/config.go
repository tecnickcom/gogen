package awssecretcache

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awssm "github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/tecnickcom/gogen/pkg/awsopt"
)

// SecretsManagerClient represents the mockable functions in the AWS SDK SecretsManagerClient client.
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

// loadConfig loads the configuration for the AWS Secret Cache.
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
