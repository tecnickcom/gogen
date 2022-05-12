package s3

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/stretchr/testify/require"
)

func Test_loadConfig(t *testing.T) {
	t.Parallel()

	got, err := loadConfig(
		context.TODO(),
		WithEndpoint("test", true),
		WithAWSOption(config.WithRegion("eu-central-1")),
	)

	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, "eu-central-1", got.Region)
}
