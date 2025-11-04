package s3

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tecnickcom/gogen/pkg/awsopt"
)

func Test_loadConfig(t *testing.T) {
	region := "eu-central-1"

	o := awsopt.Options{}
	o.WithRegion(region)

	got, err := loadConfig(
		t.Context(),
		WithAWSOptions(o),
		WithEndpointMutable("https://test.endpoint.invalid"),
	)

	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, region, got.awsConfig.Region)

	// force aws config.LoadDefaultConfig to fail
	t.Setenv("AWS_ENABLE_ENDPOINT_DISCOVERY", "ERROR")

	got, err = loadConfig(t.Context())

	require.Error(t, err)
	require.Nil(t, got)
}
