package s3

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/tecnickcom/gogen/pkg/awsopt"
)

// cfg holds the configuration for the S3 service client.
type cfg struct {
	awsConfig aws.Config
	awsOpts   awsopt.Options
	srvOptFns []SrvOptionFunc
}

// loadConfig loads the configuration for the S3 service client using the provided options.
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
