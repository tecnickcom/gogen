package sqs

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/stretchr/testify/require"
)

// nolint: paralleltest
func Test_loadConfig(t *testing.T) {
	var (
		wt  int32 = 13
		vt  int32 = 17
		reg       = "eu-central-1"
	)

	got, err := loadConfig(
		context.TODO(),
		WithEndpoint("test", true),
		WithAWSOption(config.WithRegion(reg)),
		WithWaitTimeSeconds(wt),
		WithVisibilityTimeout(vt),
	)

	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, reg, got.awsConfig.Region)
	require.Equal(t, wt, got.waitTimeSeconds)
	require.Equal(t, vt, got.visibilityTimeout)

	got, err = loadConfig(
		context.TODO(),
		WithWaitTimeSeconds(-1),
	)

	require.Error(t, err)
	require.Nil(t, got)

	got, err = loadConfig(
		context.TODO(),
		WithWaitTimeSeconds(21),
	)

	require.Error(t, err)
	require.Nil(t, got)

	got, err = loadConfig(
		context.TODO(),
		WithVisibilityTimeout(-1),
	)

	require.Error(t, err)
	require.Nil(t, got)

	got, err = loadConfig(
		context.TODO(),
		WithVisibilityTimeout(43201),
	)

	require.Error(t, err)
	require.Nil(t, got)

	// force aws config.LoadDefaultConfig to fail
	t.Setenv("AWS_ENABLE_ENDPOINT_DISCOVERY", "ERROR")

	got, err = loadConfig(context.TODO())

	require.Error(t, err)
	require.Nil(t, got)
}
