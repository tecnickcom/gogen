package s3

import (
	"context"
	"reflect"
	"testing"

	"github.com/Vonage/gosrvlib/pkg/awsopt"
	"github.com/aws/aws-sdk-go-v2/config"
	awssrv "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/require"
)

func Test_WithAWSOptions(t *testing.T) {
	t.Parallel()

	region := "ap-southeast-2"

	opt := awsopt.Options{}
	opt.WithRegion(region)

	c := &cfg{}
	gotFn := WithAWSOptions(opt)

	gotFn(c)

	want := &cfg{awsOpts: awsopt.Options{config.WithRegion(region)}}

	require.Equal(t, len(want.awsOpts), len(c.awsOpts))

	for i, opt := range want.awsOpts {
		reflect.DeepEqual(opt, c.awsOpts[i])
	}
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

			ep, err := er.ResolveEndpoint(context.TODO(), awssrv.EndpointParameters{})

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
