package s3

import (
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/stretchr/testify/require"
)

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
