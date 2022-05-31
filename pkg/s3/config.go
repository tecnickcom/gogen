package s3

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/nexmoinc/gosrvlib/pkg/awsopt"
)

type cfg struct {
	awsOpts   awsopt.Options
	awsConfig aws.Config
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
