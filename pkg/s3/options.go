package s3

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

// Option is a type to allow setting custom client options.
type Option func(*awsConfig)

func withAWSOption(opt func(*config.LoadOptions) error) Option {
	return func(cfg *awsConfig) {
		cfg.awsOpts = append(cfg.awsOpts, opt)
	}
}

// WithEndpoint overrides the AWS endpoint for the service.
func WithEndpoint(url string, isImmutable bool) Option {
	return withAWSOption(config.WithEndpointResolverWithOptions(endpointResolver{url: url, isImmutable: isImmutable}))
}

type endpointResolver struct {
	url         string
	isImmutable bool
}

func (r endpointResolver) ResolveEndpoint(_, _ string, _ ...interface{}) (aws.Endpoint, error) {
	return aws.Endpoint{URL: r.url, HostnameImmutable: r.isImmutable}, nil
}
