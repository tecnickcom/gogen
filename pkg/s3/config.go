package s3

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

type awsConfig struct {
	awsOpts []func(*config.LoadOptions) error
}

func loadConfig(ctx context.Context, opts ...Option) (aws.Config, error) {
	cfg := &awsConfig{}

	for _, apply := range opts {
		apply(cfg)
	}

	return config.LoadDefaultConfig(ctx, cfg.awsOpts...) // nolint: wrapcheck
}
