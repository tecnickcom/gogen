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
	tests := []struct {
		name                string
		url                 string
		defaultRegion       string
		envAWSRegion        string
		envAWSDefaultregion string
		wantRegion          string
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
			name:          "amazonaws.com not at the end of the host falls back",
			url:           "https://sqs.us-east-1.amazonaws.com.evil.example.com",
			defaultRegion: "sa-east-1",
			wantRegion:    "sa-east-1",
		},
		{
			name:          "Legacy global endpoint without region falls back",
			url:           "https://s3.amazonaws.com",
			defaultRegion: "us-east-1",
			wantRegion:    "us-east-1",
		},
		{
			name:          "Load default region",
			url:           "https://no-region-2.with-default.example.com",
			defaultRegion: "ap-southeast-2",
			wantRegion:    "ap-southeast-2",
		},
		{
			name:          "Load from AWS_REGION",
			url:           "https://no-region-3.example.com",
			defaultRegion: "",
			envAWSRegion:  "eu-central-1",
			wantRegion:    "eu-central-1",
		},
		{
			name:                "Load from AWS_DEFAULT_REGION",
			url:                 "https://no-region-4.example.com",
			defaultRegion:       "",
			envAWSDefaultregion: "eu-west-1",
			wantRegion:          "eu-west-1",
		},
		{
			name:          "Invalid AWS URL without default region",
			url:           "https://no-region.without-default.example.com",
			defaultRegion: "",
			wantRegion:    awsDefaultRegion,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("AWS_REGION", tt.envAWSRegion)
			t.Setenv("AWS_DEFAULT_REGION", tt.envAWSDefaultregion)

			c := Options{}
			c.WithRegionFromURL(tt.url, tt.defaultRegion)

			require.Len(t, c, 1)
			require.Equal(t, tt.wantRegion, regionFromOptions(t, c))
		})
	}
}
