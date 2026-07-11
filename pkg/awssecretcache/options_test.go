package awssecretcache

import (
	"context"
	"errors"
	"testing"
	"time"

	awssm "github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/stretchr/testify/require"
	"github.com/tecnickcom/nurago/pkg/awsopt"
)

func Test_WithAWSOptions(t *testing.T) {
	t.Parallel()

	region := "ap-southeast-2"

	opt := awsopt.Options{}
	opt.WithRegion(region)

	c := &cfg{}
	WithAWSOptions(opt)(c)

	require.Len(t, c.awsOpts, 1)

	// Assert the option's observable effect rather than its function identity
	// (LoadOptionsFunc values are not meaningfully comparable): the AWS config
	// loaded from these options must carry the configured region.
	got, err := loadConfig(t.Context(), WithAWSOptions(opt))
	require.NoError(t, err)
	require.Equal(t, region, got.awsConfig.Region)
}

func Test_WithSecretsManagerClient(t *testing.T) {
	t.Parallel()

	smclient := &mockSecretsManagerClient{
		getSecretValue: func(_ context.Context, _ *awssm.GetSecretValueInput, _ ...func(*awssm.Options)) (*awssm.GetSecretValueOutput, error) {
			return nil, errors.New("error")
		},
	}

	conf := &cfg{}
	WithSecretsManagerClient(smclient)(conf)
	require.NotEmpty(t, conf.smclient)
}

func Test_WithEndpointMutable(t *testing.T) {
	t.Parallel()

	url := "test.url.invalid"

	conf := &cfg{}
	WithEndpointMutable(url)(conf)
	require.NotEmpty(t, conf.srvOptFns)
}

func Test_WithEndpointImmutable(t *testing.T) {
	t.Parallel()

	url := "test.url.invalid"

	conf := &cfg{}
	WithEndpointImmutable(url)(conf)
	require.NotEmpty(t, conf.srvOptFns)
}

func Test_ResolveEndpoint(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{
			name:    "parse error",
			url:     "~@:;:#~",
			wantErr: true,
		},
		{
			name:    "ok",
			url:     "http://test.url.invalid",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			er := &endpointResolver{
				url: tt.url,
			}

			ep, err := er.ResolveEndpoint(t.Context(), awssm.EndpointParameters{})

			if tt.wantErr {
				require.Error(t, err)
				require.Empty(t, ep)
			} else {
				require.NoError(t, err)
				require.NotEmpty(t, ep)
			}
		})
	}
}

func Test_WithStaleIfError(t *testing.T) {
	t.Parallel()

	conf := &cfg{}

	WithStaleIfError(30 * time.Second)(conf)
	require.Equal(t, 30*time.Second, conf.maxStale)

	WithStaleIfError(0)(conf)
	require.Equal(t, time.Duration(0), conf.maxStale)
}
