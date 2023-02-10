package sqs

import (
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/stretchr/testify/require"
	"github.com/vonage/gosrvlib/pkg/awsopt"
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
