/*
Package awsopt solves the repetitive wiring problem of configuring the
aws-sdk-go-v2 library consistently across multiple AWS service clients. Rather
than scattering [config.LoadOptionsFunc] calls throughout every package that
talks to AWS, awsopt centralizes those options into a single composable
[Options] slice that can be built once and handed to any AWS-based package in
this library.

# Problem

Every aws-sdk-go-v2 client requires a loaded [aws.Config], and every project
ends up writing the same region-detection boilerplate: check a hardcoded value,
fall back to an environment variable, fall back to a default. When several
packages (secrets, S3, SQS, …) each duplicate this logic, configuration drift
and inconsistent behavior are inevitable. awsopt provides one canonical
implementation shared by all of them.

# How It Works

[Options] is a typed slice of [config.LoadOptionsFunc] values. Options are
accumulated with method calls and then materialized into an [aws.Config] via
[Options.LoadDefaultConfig], which delegates to the standard
[config.LoadDefaultConfig] from the SDK:

 1. Create an [Options] value (zero value is ready to use).
 2. Append options with [Options.WithAWSOption], [Options.WithRegion], or
    [Options.WithRegionFromURL].
 3. Call [Options.LoadDefaultConfig] to obtain an [aws.Config] that can be
    passed directly to any aws-sdk-go-v2 service constructor.

# Key Features

  - Composable options: [Options] is an append-friendly slice; options from
    multiple sources can be merged before the config is loaded.
  - Automatic region resolution: [Options.WithRegionFromURL] extracts the
    AWS region from a standard amazonaws.com service endpoint URL
    (protocol://service-code.region-code.amazonaws.com). When the URL does not
    encode a region, it falls back through a well-defined precedence chain:
    caller-supplied default → AWS_REGION env var → AWS_DEFAULT_REGION env var
    → the built-in constant (us-east-2). This single function eliminates an
    entire class of misconfiguration bugs.
  - Escape hatch: [Options.WithAWSOption] accepts any [config.LoadOptionsFunc],
    so every SDK option remains accessible without leaving the awsopt pattern.
  - Zero dependencies beyond aws-sdk-go-v2: the package adds no third-party
    dependencies of its own.

# Integrated Packages

awsopt is used as the AWS configuration layer by:
  - [github.com/tecnickcom/gogen/pkg/awssecretcache]
  - [github.com/tecnickcom/gogen/pkg/s3]
  - [github.com/tecnickcom/gogen/pkg/sqs]

# Usage

	var opts awsopt.Options
	opts.WithRegionFromURL("https://s3.eu-west-1.amazonaws.com", "")

	awsCfg, err := opts.LoadDefaultConfig(ctx)
	if err != nil {
	    return err
	}

	// awsCfg is ready for any aws-sdk-go-v2 service client:
	// s3.NewFromConfig(awsCfg), secretsmanager.NewFromConfig(awsCfg), …

When consuming an awsopt-based package, pass additional options via the
package's own WithAWSOptions helper so all AWS clients in the process share
consistent configuration.
*/
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

// LoadDefaultConfig builds an aws.Config from the accumulated options.
//
// It solves the "repeat config wiring in every AWS client" problem by turning
// one shared Options value into a concrete SDK configuration in a single call.
// This keeps region, credentials, and any custom SDK loaders consistent across
// packages that consume awsopt.
func (c *Options) LoadDefaultConfig(ctx context.Context) (aws.Config, error) {
	o := make([]func(*config.LoadOptions) error, len(*c))
	for k, v := range *c {
		o[k] = (func(*config.LoadOptions) error)(v)
	}

	return config.LoadDefaultConfig(ctx, o...) //nolint:wrapcheck
}

// WithAWSOption appends any raw aws-sdk-go-v2 load option.
//
// Use this as the escape hatch when you need an SDK feature that does not have
// a dedicated helper in this package. The main benefit is flexibility without
// giving up the shared Options composition pattern.
func (c *Options) WithAWSOption(opt config.LoadOptionsFunc) {
	*c = append(*c, opt)
}

// WithRegion appends an explicit AWS region option.
//
// This is the direct choice when the region is already known, and it ensures
// every downstream AWS client built from the same Options value uses the same
// target region.
func (c *Options) WithRegion(region string) {
	c.WithAWSOption(config.WithRegion(region))
}

// WithRegionFromURL appends a region derived from an AWS endpoint URL.
//
// It removes common endpoint-parsing boilerplate by extracting the region from
// URLs like "https://service.region.amazonaws.com" and automatically applying
// robust fallbacks. If the URL does not contain a region, a default region is
// selected with the following order of precedence:
//   - the specified defaultRegion;
//   - the AWS_REGION environment variable;
//   - the AWS_DEFAULT_REGION environment variable;
//   - the region set in the awsDefaultRegion constant.
//
// The benefit is predictable region resolution across services, even when input
// URLs are incomplete or environment-specific.
func (c *Options) WithRegionFromURL(url, defaultRegion string) {
	c.WithRegion(awsRegionFromURL(url, defaultRegion))
}

// awsRegionFromURL extracts an AWS region from a service endpoint URL.
//
// When extraction fails, it returns the first available fallback region from
// the precedence chain documented by WithRegionFromURL.
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
