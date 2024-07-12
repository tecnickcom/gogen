package awssecretcache

import (
	"context"
	"fmt"

	"github.com/Vonage/gosrvlib/pkg/awsopt"
	"github.com/aws/aws-sdk-go-v2/aws"
	awssm "github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

// SecretsManagerClient represents the mockable functions in the AWS SDK SecretsManagerClient client.
type SecretsManagerClient interface {
	GetSecretValue(ctx context.Context, params *awssm.GetSecretValueInput, optFns ...func(*awssm.Options)) (*awssm.GetSecretValueOutput, error)
}

type cfg struct {
	awsOpts   awsopt.Options
	awsConfig aws.Config
	srvOptFns []SrvOptionFunc
	smclient  SecretsManagerClient
}

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
