package sqs

import (
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/stretchr/testify/require"
)

func Test_WithWaitTimeSeconds(t *testing.T) {
	t.Parallel()

	var v int32 = 13

	conf := &cfg{}
	WithWaitTimeSeconds(v)(conf)
	require.Equal(t, v, conf.waitTimeSeconds)
}

func Test_WithVisibilityTimeout(t *testing.T) {
	t.Parallel()

	var v int32 = 17

	conf := &cfg{}
	WithVisibilityTimeout(v)(conf)
	require.Equal(t, v, conf.visibilityTimeout)
}

func Test_WithRegion(t *testing.T) {
	t.Parallel()

	region := "ap-southeast-2"

	c := &cfg{}
	gotFn := WithRegion(region)

	gotFn(c)

	want := &cfg{awsOpts: []func(*config.LoadOptions) error{config.WithRegion(region)}}

	require.Equal(t, len(want.awsOpts), len(c.awsOpts))

	for i, opt := range want.awsOpts {
		reflect.DeepEqual(opt, c.awsOpts[i])
	}
}

// nolint:paralleltest,tparallel
func Test_WithRegionFromURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		url                 string
		defaultRegion       string
		envAWSRegion        string
		envAWSDefaultregion string
		want                *cfg
	}{
		{
			name: "Valid AWS URL",
			url:  "https://sqs.ap-southeast-1.amazonaws.com",
			want: &cfg{awsOpts: []func(*config.LoadOptions) error{config.WithRegion("ap-southeast-1")}},
		},
		{
			name:          "Load default region",
			url:           "https://no-region-2.with-default.example.com",
			defaultRegion: "ap-southeast-2",
			want:          &cfg{awsOpts: []func(*config.LoadOptions) error{config.WithRegion("ap-southeast-2")}},
		},
		{
			name:          "Load from AWS_REGION",
			url:           "https://no-region-3.example.com",
			defaultRegion: "",
			envAWSRegion:  "eu-central-1",
			want:          &cfg{awsOpts: []func(*config.LoadOptions) error{config.WithRegion("eu-central-1")}},
		},
		{
			name:                "Load from AWS_DEFAULT_REGION",
			url:                 "https://no-region-4.example.com",
			defaultRegion:       "",
			envAWSDefaultregion: "eu-west-1",
			want:                &cfg{awsOpts: []func(*config.LoadOptions) error{config.WithRegion("eu-west-1")}},
		},
		{
			name:          "Invalid AWS URL without default region",
			url:           "https://no-region.without-default.example.com",
			defaultRegion: "",
			want:          &cfg{awsOpts: []func(*config.LoadOptions) error{config.WithRegion(awsDefaultRegion)}},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("AWS_REGION", tt.envAWSRegion)
			t.Setenv("AWS_DEFAULT_REGION", tt.envAWSDefaultregion)

			c := &cfg{}
			gotFn := WithRegionFromURL(tt.url, tt.defaultRegion)

			gotFn(c)

			require.Equal(t, len(tt.want.awsOpts), len(c.awsOpts))

			for i, opt := range tt.want.awsOpts {
				reflect.DeepEqual(opt, c.awsOpts[i])
			}
		})
	}
}

func Test_WithEndpoint(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		url       string
		immutable bool
		want      *cfg
	}{
		{
			name:      "Immutable URL",
			url:       "test_a",
			immutable: true,
			want: &cfg{awsOpts: []func(*config.LoadOptions) error{
				config.WithEndpointResolverWithOptions(endpointResolver{url: "test_a", isImmutable: true})},
			},
		},
		{
			name:      "Mutable URL",
			url:       "test_b",
			immutable: false,
			want: &cfg{awsOpts: []func(*config.LoadOptions) error{
				config.WithEndpointResolverWithOptions(endpointResolver{url: "test_b", isImmutable: false})},
			},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := &cfg{}
			gotFn := WithEndpoint(tt.url, tt.immutable)

			gotFn(c)

			require.Equal(t, len(tt.want.awsOpts), len(c.awsOpts))

			for i, opt := range tt.want.awsOpts {
				reflect.DeepEqual(opt, c.awsOpts[i])
			}
		})
	}
}

func Test_ResolveEndpoint(t *testing.T) {
	t.Parallel()

	er := &endpointResolver{
		url:         "test_url",
		isImmutable: true,
	}

	ep, err := er.ResolveEndpoint("", "", nil)
	require.NoError(t, err)
	require.Equal(t, er.url, ep.URL)
	require.Equal(t, er.isImmutable, ep.HostnameImmutable)
}
