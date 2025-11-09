package awsopt

import (
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/stretchr/testify/require"
)

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
	want := Options{config.WithRegion(region)}

	c := Options{}
	c.WithAWSOption(config.WithRegion(region))

	require.Len(t, c, len(want))

	for i, opt := range want {
		reflect.DeepEqual(opt, c[i])
	}
}

func Test_WithRegion(t *testing.T) {
	t.Parallel()

	region := "eu-central-1"
	want := Options{config.WithRegion(region)}

	c := Options{}
	c.WithRegion(region)

	require.Len(t, c, len(want))

	for i, opt := range want {
		reflect.DeepEqual(opt, c[i])
	}
}

func Test_WithRegionFromURL(t *testing.T) {
	tests := []struct {
		name                string
		url                 string
		defaultRegion       string
		envAWSRegion        string
		envAWSDefaultregion string
		want                Options
	}{
		{
			name: "Valid AWS URL",
			url:  "https://sqs.ap-southeast-1.amazonaws.com",
			want: Options{config.WithRegion("ap-southeast-1")},
		},
		{
			name: "Valid AWS URL with custom service",
			url:  "https://some-service.af-south-1.amazonaws.com",
			want: Options{config.WithRegion("af-south-1")},
		},
		{
			name:          "Load default region",
			url:           "https://no-region-2.with-default.example.com",
			defaultRegion: "ap-southeast-2",
			want:          Options{config.WithRegion("ap-southeast-2")},
		},
		{
			name:          "Load from AWS_REGION",
			url:           "https://no-region-3.example.com",
			defaultRegion: "",
			envAWSRegion:  "eu-central-1",
			want:          Options{config.WithRegion("eu-central-1")},
		},
		{
			name:                "Load from AWS_DEFAULT_REGION",
			url:                 "https://no-region-4.example.com",
			defaultRegion:       "",
			envAWSDefaultregion: "eu-west-1",
			want:                Options{config.WithRegion("eu-west-1")},
		},
		{
			name:          "Invalid AWS URL without default region",
			url:           "https://no-region.without-default.example.com",
			defaultRegion: "",
			want:          Options{config.WithRegion(awsDefaultRegion)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("AWS_REGION", tt.envAWSRegion)
			t.Setenv("AWS_DEFAULT_REGION", tt.envAWSDefaultregion)

			c := Options{}
			c.WithRegionFromURL(tt.url, tt.defaultRegion)

			require.Len(t, c, len(tt.want))

			for i, opt := range tt.want {
				reflect.DeepEqual(opt, c[i])
			}
		})
	}
}
