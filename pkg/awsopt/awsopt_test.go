package awsopt

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/stretchr/testify/require"
)

// regionFromOptions applies the accumulated options to a config.LoadOptions
// value and returns the resulting region, so tests can assert each option's
// effect rather than comparing opaque function values.
func regionFromOptions(t *testing.T, opts Options) string {
	t.Helper()

	var lo config.LoadOptions

	for _, opt := range opts {
		require.NoError(t, opt(&lo))
	}

	return lo.Region
}

func Test_LoadDefaultConfig(t *testing.T) {
	region := "us-west-2"

	c := Options{}
	c.WithAWSOption(config.WithRegion(region))

	got, err := c.LoadDefaultConfig(t.Context())

	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, region, got.Region)

	// force aws config.LoadDefaultConfig to fail
	t.Setenv("AWS_ENABLE_ENDPOINT_DISCOVERY", "ERROR")

	_, err = c.LoadDefaultConfig(t.Context())

	require.Error(t, err)
}

func Test_LoadDefaultConfig_ZeroValue(t *testing.T) {
	// A zero-value Options is documented as ready to use. Neutralize any ambient
	// AWS_ENABLE_ENDPOINT_DISCOVERY so the result is deterministic.
	t.Setenv("AWS_ENABLE_ENDPOINT_DISCOVERY", "")

	var c Options

	got, err := c.LoadDefaultConfig(t.Context())

	require.NoError(t, err)
	require.NotNil(t, got)
}

func Test_WithAWSOption(t *testing.T) {
	t.Parallel()

	region := "ap-southeast-2"

	c := Options{}
	c.WithAWSOption(config.WithRegion(region))

	require.Len(t, c, 1)
	require.Equal(t, region, regionFromOptions(t, c))
}

func Test_WithRegion(t *testing.T) {
	t.Parallel()

	region := "eu-central-1"

	c := Options{}
	c.WithRegion(region)

	require.Len(t, c, 1)
	require.Equal(t, region, regionFromOptions(t, c))
}

func Test_WithRegionFromURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		url           string
		defaultRegion string
		wantRegion    string // "" means no region option is added (resolution deferred to the SDK)
	}{
		{
			name:       "Valid AWS URL",
			url:        "https://sqs.ap-southeast-1.amazonaws.com",
			wantRegion: "ap-southeast-1",
		},
		{
			name:       "Valid AWS URL with custom service",
			url:        "https://some-service.af-south-1.amazonaws.com",
			wantRegion: "af-south-1",
		},
		{
			name:       "Valid AWS URL with http scheme (localstack)",
			url:        "http://sqs.eu-west-1.amazonaws.com",
			wantRegion: "eu-west-1",
		},
		{
			name:       "Valid AWS URL without scheme",
			url:        "sqs.eu-west-2.amazonaws.com",
			wantRegion: "eu-west-2",
		},
		{
			name:       "Valid virtual-hosted-style S3 URL",
			url:        "https://bucket.s3.eu-west-1.amazonaws.com",
			wantRegion: "eu-west-1",
		},
		{
			name:       "Valid AWS URL with port and path",
			url:        "https://sqs.us-west-2.amazonaws.com:443/123456789012/my-queue",
			wantRegion: "us-west-2",
		},
		{
			name:       "China partition endpoint (amazonaws.com.cn)",
			url:        "https://sqs.cn-north-1.amazonaws.com.cn",
			wantRegion: "cn-north-1",
		},
		{
			name:       "GovCloud region",
			url:        "https://sqs.us-gov-west-1.amazonaws.com",
			wantRegion: "us-gov-west-1",
		},
		{
			name:       "Dual-stack endpoint on api.aws (not amazonaws.com)",
			url:        "https://ec2.us-west-2.api.aws",
			wantRegion: "us-west-2",
		},
		{
			name:       "ISO partition on its own domain (c2s.ic.gov)",
			url:        "https://sqs.us-iso-east-1.c2s.ic.gov",
			wantRegion: "us-iso-east-1",
		},
		{
			name:       "ISO-B partition on its own domain (sc2s.sgov.gov)",
			url:        "https://sqs.us-isob-east-1.sc2s.sgov.gov",
			wantRegion: "us-isob-east-1",
		},
		{
			name:       "S3 dual-stack endpoint on amazonaws.com",
			url:        "https://s3.dualstack.eu-west-1.amazonaws.com",
			wantRegion: "eu-west-1",
		},
		{
			name:       "Mixed-case host is normalized",
			url:        "https://SQS.US-EAST-1.AMAZONAWS.COM",
			wantRegion: "us-east-1",
		},
		{
			name:       "VPC interface endpoint (region sits before the vpce label)",
			url:        "https://vpce-0abc-1234.sqs.us-east-1.vpce.amazonaws.com",
			wantRegion: "us-east-1",
		},
		{
			name:       "VPC zonal interface endpoint",
			url:        "https://vpce-0e1bb1e4-abcdefgh-us-east-1a.ec2.us-east-1.vpce.amazonaws.com",
			wantRegion: "us-east-1",
		},
		{
			name:       "Region-shaped leading label does not shadow the real region",
			url:        "https://us-west-2.s3.eu-west-1.amazonaws.com",
			wantRegion: "eu-west-1",
		},
		{
			name:       "Region-shaped label is extracted even on a non-AWS host",
			url:        "https://sqs.us-east-1.amazonaws.com.evil.example.com",
			wantRegion: "us-east-1",
		},
		{
			name:       "S3 website dot-style endpoint",
			url:        "https://bucket.s3-website.eu-west-1.amazonaws.com",
			wantRegion: "eu-west-1",
		},
		{
			name:       "Scheme-less URL with :// in the query",
			url:        "sqs.us-east-1.amazonaws.com/redir?u=http://example.com",
			wantRegion: "us-east-1",
		},
		{
			name:          "Global endpoint without a region label falls back",
			url:           "https://s3.amazonaws.com",
			defaultRegion: "us-east-1",
			wantRegion:    "us-east-1",
		},
		{
			name:          "No region in URL uses default region",
			url:           "https://service.example.com",
			defaultRegion: "ap-southeast-2",
			wantRegion:    "ap-southeast-2",
		},
		{
			name:          "Empty URL uses default region",
			url:           "",
			defaultRegion: "eu-north-1",
			wantRegion:    "eu-north-1",
		},
		{
			name:          "Malformed URL (invalid port) uses default region",
			url:           "https://sqs.eu-west-1.amazonaws.com:notaport",
			defaultRegion: "eu-north-1",
			wantRegion:    "eu-north-1",
		},
		{
			name:       "No region and no default adds no option (SDK resolves)",
			url:        "https://service.example.com",
			wantRegion: "",
		},
		{
			name:       "URL path segment is not mistaken for a region",
			url:        "https://service.example.com/us-east-1/queue",
			wantRegion: "",
		},
		{
			name:       "Malformed URL without default adds no option",
			url:        "https://sqs.eu-west-1.amazonaws.com:notaport",
			wantRegion: "",
		},
		{
			name:       "Scheme-less URL with an invalid port adds no option",
			url:        "sqs.eu-west-1.amazonaws.com:notaport",
			wantRegion: "",
		},
		{
			name:       "IPv4 host is not a region (localstack)",
			url:        "https://127.0.0.1:4566",
			wantRegion: "",
		},
		{
			name:       "IPv6 host is not a region",
			url:        "https://[::1]:4566",
			wantRegion: "",
		},
		{
			name:       "localhost is not a region",
			url:        "https://localhost:4566",
			wantRegion: "",
		},
		{
			name:       "S3 website dash-style endpoint is not recognized (region folded into label)",
			url:        "https://bucket.s3-website-us-east-1.amazonaws.com",
			wantRegion: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := Options{}
			c.WithRegionFromURL(tt.url, tt.defaultRegion)

			if tt.wantRegion == "" {
				require.Empty(t, c)

				return
			}

			require.Len(t, c, 1)
			require.Equal(t, tt.wantRegion, regionFromOptions(t, c))
		})
	}
}
