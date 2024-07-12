package awssecretcache

import (
	"context"
	"testing"

	"github.com/Vonage/gosrvlib/pkg/awsopt"
	"github.com/stretchr/testify/require"
)

func Test_loadConfig(t *testing.T) {
	region := "eu-central-1"

	o := awsopt.Options{}
	o.WithRegion(region)
	// o.WithEndpoint("https://test.endpoint.invalid", true) // deprecated

	got, err := loadConfig(
		context.TODO(),
		WithAWSOptions(o),
		WithEndpointMutable("https://test.endpoint.invalid"),
	)

	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, region, got.awsConfig.Region)

	// force aws config.LoadDefaultConfig to fail
	t.Setenv("AWS_ENABLE_ENDPOINT_DISCOVERY", "ERROR")

	got, err = loadConfig(context.TODO())

	require.Error(t, err)
	require.Nil(t, got)
}
