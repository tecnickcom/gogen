package sqs

import (
	"os"
	"regexp"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

const (
	awsRegionFromURLRegexp = `^https://sqs.([^.]+).amazonaws.com` // protocol://service-code.region-code.amazonaws.com
	awsDefaultRegion       = "us-east-2"
)

// Option is a type to allow setting custom client options.
type Option func(*cfg)

// WithWaitTimeSeconds overrides the default duration (in seconds) for which the call waits for a message to arrive in the queue before returning.
// Values range: 0 to 20 seconds.
func WithWaitTimeSeconds(t int32) Option {
	return func(c *cfg) {
		c.waitTimeSeconds = t
	}
}

// WithVisibilityTimeout overrides the default duration (in seconds) that the received messages are hidden from subsequent retrieve requests after being retrieved by a ReceiveMessage request.
// Values range: 0 to 43200. Maximum: 12 hours.
func WithVisibilityTimeout(t int32) Option {
	return func(c *cfg) {
		c.visibilityTimeout = t
	}
}

// WithRegion allows to specify the AWS region.
func WithRegion(region string) Option {
	return WithAWSOption(config.WithRegion(region))
}

// WithRegionFromURL allows to specify the AWS region extracted from the provided URL.
// If the URL does not contain a region, a default one will be returned with the order of precedence:
//   * the specified defaultRegion;
//   * the AWS_REGION environment variable;
//   * the AWS_DEFAULT_REGION environment variable;
//   * the region set in the awsDefaultRegion constant.
func WithRegionFromURL(url, defaultRegion string) Option {
	return WithRegion(awsRegionFromURL(url, defaultRegion))
}

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

// WithAWSOption allows to add an arbitrary AWS option.
func WithAWSOption(opt func(*config.LoadOptions) error) Option {
	return func(c *cfg) {
		c.awsOpts = append(c.awsOpts, opt)
	}
}

// WithEndpoint overrides the AWS endpoint for the service.
func WithEndpoint(url string, isImmutable bool) Option {
	return WithAWSOption(config.WithEndpointResolverWithOptions(&endpointResolver{
		url:         url,
		isImmutable: isImmutable,
	}))
}

type endpointResolver struct {
	url         string
	isImmutable bool
}

func (r endpointResolver) ResolveEndpoint(_, region string, _ ...interface{}) (aws.Endpoint, error) {
	return aws.Endpoint{
		SigningRegion:     region,
		URL:               r.url,
		HostnameImmutable: r.isImmutable,
	}, nil
}
