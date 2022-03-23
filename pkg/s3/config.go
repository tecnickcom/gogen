package s3

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

type cfg struct {
	awsOpts []func(*config.LoadOptions) error
}

func loadConfig(ctx context.Context, opts ...Option) (aws.Config, error) {
	c := &cfg{}

	for _, apply := range opts {
		apply(c)
	}

	return config.LoadDefaultConfig(ctx, c.awsOpts...) // nolint: wrapcheck
}
