package sqs

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tecnickcom/gogen/pkg/awsopt"
)

func Test_loadConfig(t *testing.T) {
	var (
		wt     int32 = 13
		vt     int32 = 17
		region       = "eu-central-1"
	)

	o := awsopt.Options{}
	o.WithRegion(region)

	got, err := loadConfig(
		t.Context(),
		"https://test_queue.invalid/queue0.fifo",
		WithAWSOptions(o),
		WithEndpointMutable("https://test.endpoint.invalid"),
		WithWaitTimeSeconds(wt),
		WithVisibilityTimeout(vt),
	)

	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, region, got.awsConfig.Region)
	require.Equal(t, wt, got.waitTimeSeconds)
	require.Equal(t, vt, got.visibilityTimeout)
	require.NotNil(t, got.messageEncodeFunc)
	require.NotNil(t, got.messageDecodeFunc)

	// region is derived from the queue URL when no explicit region is set
	got, err = loadConfig(
		t.Context(),
		"https://sqs.ap-south-1.amazonaws.com/123456789012/my-queue",
	)

	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, "ap-south-1", got.awsConfig.Region)

	// an explicit region overrides the one derived from the queue URL
	oOverride := awsopt.Options{}
	oOverride.WithRegion(region)

	got, err = loadConfig(
		t.Context(),
		"https://sqs.ap-south-1.amazonaws.com/123456789012/my-queue",
		WithAWSOptions(oOverride),
	)

	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, region, got.awsConfig.Region)

	got, err = loadConfig(
		t.Context(),
		"https://test_queue.invalid/queue0.fifo",
		WithMessageEncodeFunc(nil),
	)

	require.Error(t, err)
	require.Nil(t, got)

	got, err = loadConfig(
		t.Context(),
		"https://test_queue.invalid/queue0.fifo",
		WithMessageDecodeFunc(nil),
	)

	require.Error(t, err)
	require.Nil(t, got)

	got, err = loadConfig(
		t.Context(),
		"https://test_queue.invalid/queue0.fifo",
		WithWaitTimeSeconds(-1),
	)

	require.Error(t, err)
	require.Nil(t, got)

	got, err = loadConfig(
		t.Context(),
		"https://test_queue.invalid/queue0.fifo",
		WithWaitTimeSeconds(21),
	)

	require.Error(t, err)
	require.Nil(t, got)

	got, err = loadConfig(
		t.Context(),
		"https://test_queue.invalid/queue0.fifo",
		WithVisibilityTimeout(-1),
	)

	require.Error(t, err)
	require.Nil(t, got)

	got, err = loadConfig(
		t.Context(),
		"https://test_queue.invalid/queue0.fifo",
		WithVisibilityTimeout(43201),
	)

	require.Error(t, err)
	require.Nil(t, got)

	// force aws config.LoadDefaultConfig to fail
	t.Setenv("AWS_ENABLE_ENDPOINT_DISCOVERY", "ERROR")

	got, err = loadConfig(t.Context(), "https://test_queue.invalid/queue0.fifo")

	require.Error(t, err)
	require.Nil(t, got)

	// an injected client skips AWS config loading, so it succeeds even with the
	// failing environment above still set
	got, err = loadConfig(
		t.Context(),
		"https://test_queue.invalid/queue0.fifo",
		WithSQSClient(sqsmock{}),
	)

	require.NoError(t, err)
	require.NotNil(t, got)
	require.NotNil(t, got.sqsClient)
}
