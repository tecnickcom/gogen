// Package awsopt provides functions to configure common AWS options.
package awsopt

import (
	"context"
	"os"
	"regexp"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

const (
	// awsRegionFromURLRegexp is a regular expression used to extract the region from URL.
	// protocol://service-code.region-code.amazonaws.com
	awsRegionFromURLRegexp = `^https://[^\.]+\.([^\.]+)\.amazonaws\.com`

	// awsDefaultRegion is the region that will be used if any other way to detect the region fails.
	awsDefaultRegion = "us-east-2"
)

// Options is a set of all AWS options to apply.
type Options []config.LoadOptionsFunc

// LoadDefaultConfig populates an AWS Config with the values from the external configurations and set options.
func (c *Options) LoadDefaultConfig(ctx context.Context) (aws.Config, error) {
	o := make([]func(*config.LoadOptions) error, len(*c))
	for k, v := range *c {
		o[k] = (func(*config.LoadOptions) error)(v)
	}

	return config.LoadDefaultConfig(ctx, o...) //nolint:wrapcheck
}

// WithAWSOption allows to add an arbitrary AWS option.
func (c *Options) WithAWSOption(opt config.LoadOptionsFunc) {
	*c = append(*c, opt)
}

// WithRegion allows to specify the AWS region.
func (c *Options) WithRegion(region string) {
	c.WithAWSOption(config.WithRegion(region))
}

// WithRegionFromURL allows to specify the AWS region extracted from the provided URL.
// If the URL does not contain a region, a default one will be returned with the order of precedence:
//   - the specified defaultRegion;
//   - the AWS_REGION environment variable;
//   - the AWS_DEFAULT_REGION environment variable;
//   - the region set in the awsDefaultRegion constant.
func (c *Options) WithRegionFromURL(url, defaultRegion string) {
	c.WithRegion(awsRegionFromURL(url, defaultRegion))
}

// awsRegionFromURL extracts a region from a URL string or return the default value.
func awsRegionFromURL(url, defaultRegion string) string {
	re := regexp.MustCompile(awsRegionFromURLRegexp)
	match := re.FindStringSubmatch(url)

	if len(match) > 1 {
		return match[1]
	}

	if defaultRegion != "" {
		return defaultRegion
	}

	r := os.Getenv("AWS_REGION")
	if r != "" {
		return r
	}

	r = os.Getenv("AWS_DEFAULT_REGION")
	if r != "" {
		return r
	}

	return awsDefaultRegion
}

// WithEndpoint overrides the AWS endpoint for the service.
func (c *Options) WithEndpoint(url string, isImmutable bool) {
	c.WithAWSOption(
		config.WithEndpointResolverWithOptions(
			&endpointResolver{
				url:         url,
				isImmutable: isImmutable,
			},
		),
	)
}

type endpointResolver struct {
	url         string
	isImmutable bool
}

// ResolveEndpoint returns an aws.Endpoint.
func (r endpointResolver) ResolveEndpoint(_, region string, _ ...any) (aws.Endpoint, error) {
	return aws.Endpoint{
		SigningRegion:     region,
		URL:               r.url,
		HostnameImmutable: r.isImmutable,
	}, nil
}
