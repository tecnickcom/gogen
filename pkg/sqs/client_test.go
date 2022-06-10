package sqs

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/nexmoinc/gosrvlib/pkg/awsopt"
	"github.com/stretchr/testify/require"
)

// nolint: paralleltest
func TestNew(t *testing.T) {
	var (
		wt int32 = 13
		vt int32 = 17
	)

	o := awsopt.Options{}
	o.WithRegion("eu-west-1")
	o.WithEndpoint("https://test.endpoint.invalid", true)

	got, err := New(
		context.TODO(),
		"https://test_queue.invalid/queue0.fifo",
		"",
		WithAWSOptions(o),
		WithWaitTimeSeconds(wt),
		WithVisibilityTimeout(vt),
	)

	require.Error(t, err)
	require.Nil(t, got)

	got, err = New(
		context.TODO(),
		"https://test_queue.invalid/queue1.fifo",
		"alpha beta",
		WithAWSOptions(o),
		WithWaitTimeSeconds(wt),
		WithVisibilityTimeout(vt),
	)

	require.Error(t, err)
	require.Nil(t, got)

	msgGrpID := `ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!"#$%&'()*+,\-./:;<=>?@[\\\]^_` + "`" + `{|}~`
	got, err = New(
		context.TODO(),
		"https://test_queue.invalid/queue2.fifo",
		msgGrpID,
		WithAWSOptions(o),
		WithWaitTimeSeconds(wt),
		WithVisibilityTimeout(vt),
	)

	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, aws.String("https://test_queue.invalid/queue2.fifo"), got.queueURL)
	require.Equal(t, aws.String(msgGrpID), got.messageGroupID)
	require.Equal(t, wt, got.waitTimeSeconds)
	require.Equal(t, vt, got.visibilityTimeout)

	got, err = New(
		context.TODO(),
		"https://test_queue.invalid/queue3.standard",
		"SOMETHING_TO_IGNORE",
		WithAWSOptions(o),
		WithWaitTimeSeconds(wt),
		WithVisibilityTimeout(vt),
	)

	var expMessageGroupID *string

	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, aws.String("https://test_queue.invalid/queue3.standard"), got.queueURL)
	require.Equal(t, expMessageGroupID, got.messageGroupID)
	require.Equal(t, wt, got.waitTimeSeconds)
	require.Equal(t, vt, got.visibilityTimeout)

	// make AWS lib to return an error
	t.Setenv("AWS_ENABLE_ENDPOINT_DISCOVERY", "ERROR")

	got, err = New(context.TODO(), "", "")
	require.Error(t, err)
	require.Nil(t, got)
}

type sqsmock struct {
	deleteFn             func(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error)
	getQueueAttributesFn func(ctx context.Context, params *sqs.GetQueueAttributesInput, optFns ...func(*sqs.Options)) (*sqs.GetQueueAttributesOutput, error)
	receiveFn            func(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error)
	sendFn               func(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error)
}

func (s sqsmock) DeleteMessage(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error) {
	return s.deleteFn(ctx, params, optFns...)
}

func (s sqsmock) GetQueueAttributes(ctx context.Context, params *sqs.GetQueueAttributesInput, optFns ...func(*sqs.Options)) (*sqs.GetQueueAttributesOutput, error) {
	return s.getQueueAttributesFn(ctx, params, optFns...)
}

func (s sqsmock) ReceiveMessage(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
	return s.receiveFn(ctx, params, optFns...)
}

func (s sqsmock) SendMessage(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error) {
	return s.sendFn(ctx, params, optFns...)
}

func TestSend(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mock    SQS
		wantErr bool
	}{
		{
			name: "success",
			mock: sqsmock{sendFn: func(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error) {
				return &sqs.SendMessageOutput{}, nil
			}},
			wantErr: false,
		},
		{
			name: "error",
			mock: sqsmock{sendFn: func(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error) {
				return nil, fmt.Errorf("some err")
			}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.TODO()
			cli, err := New(ctx, "https://test_queue.invalid/queue1.fifo", "TEST_MSG_GROUP_ID_1")
			require.NoError(t, err)
			require.NotNil(t, cli)

			cli.sqs = tt.mock

			err = cli.Send(ctx, "test")
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestReceive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mock    SQS
		want    *Message
		wantErr bool
	}{
		{
			name: "success",
			mock: sqsmock{receiveFn: func(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
				return &sqs.ReceiveMessageOutput{
					Messages: []types.Message{
						{
							Body:          aws.String("testBody01"),
							ReceiptHandle: aws.String("TestReceiptHandle01"),
						},
					},
				}, nil
			}},
			want: &Message{
				Body:          "testBody01",
				ReceiptHandle: "TestReceiptHandle01",
			},
			wantErr: false,
		},
		{
			name: "empty",
			mock: sqsmock{receiveFn: func(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
				return &sqs.ReceiveMessageOutput{}, nil
			}},
			want:    nil,
			wantErr: false,
		},
		{
			name: "error",
			mock: sqsmock{receiveFn: func(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
				return nil, fmt.Errorf("some err")
			}},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.TODO()
			cli, err := New(ctx, "https://test_queue.invalid/queue2.fifo", "TEST_MSG_GROUP_ID_2")
			require.NoError(t, err)
			require.NotNil(t, cli)

			cli.sqs = tt.mock

			got, err := cli.Receive(ctx)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestDelete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		receiptHandle string
		mock          SQS
		wantErr       bool
	}{
		{
			name:          "success",
			receiptHandle: "123456",
			mock: sqsmock{deleteFn: func(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error) {
				return &sqs.DeleteMessageOutput{}, nil
			}},
			wantErr: false,
		},
		{
			name:          "empty",
			receiptHandle: "",
			mock: sqsmock{deleteFn: func(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error) {
				return &sqs.DeleteMessageOutput{}, nil
			}},
			wantErr: false,
		},
		{
			name:          "error",
			receiptHandle: "7890",
			mock: sqsmock{deleteFn: func(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error) {
				return nil, fmt.Errorf("some err")
			}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.TODO()
			cli, err := New(ctx, "https://test_queue.invalid/queue3.fifo", "TEST_MSG_GROUP_ID_3")
			require.NoError(t, err)
			require.NotNil(t, cli)

			cli.sqs = tt.mock

			err = cli.Delete(ctx, tt.receiptHandle)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestMessageEncode(t *testing.T) {
	t.Parallel()

	type TestData struct {
		Alpha string
		Beta  int
	}

	got, err := MessageEncode(&TestData{Alpha: "abc123", Beta: -375})
	require.NoError(t, err)
	require.NotEmpty(t, got)

	got, err = MessageEncode(nil)
	require.Error(t, err)
	require.Equal(t, "", got)
}

func TestMessageDecode(t *testing.T) {
	t.Parallel()

	type TestData struct {
		Alpha string
		Beta  int
	}

	tests := []struct {
		name    string
		msg     string
		want    TestData
		wantErr bool
	}{
		{
			name:    "success",
			msg:     "Kf+BAwEBCFRlc3REYXRhAf+CAAECAQVBbHBoYQEMAAEEQmV0YQEEAAAAD/+CAQZhYmMxMjMB/gLtAA==",
			want:    TestData{Alpha: "abc123", Beta: -375},
			wantErr: false,
		},
		{
			name:    "invalid base64",
			msg:     "你好世界",
			want:    TestData{},
			wantErr: true,
		},
		{
			name:    "empty",
			msg:     "",
			want:    TestData{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var data TestData

			err := MessageDecode(tt.msg, &data)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.want.Alpha, data.Alpha)
			require.Equal(t, tt.want.Beta, data.Beta)
		})
	}
}

func TestSendData(t *testing.T) {
	t.Parallel()

	ctx := context.TODO()
	cli, err := New(ctx, "https://test_queue.invalid/queue4.fifo", "TEST_MSG_GROUP_ID_4")
	require.NoError(t, err)
	require.NotNil(t, cli)

	cli.sqs = sqsmock{sendFn: func(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error) {
		return &sqs.SendMessageOutput{}, nil
	}}

	type TestData struct {
		Alpha string
		Beta  int
	}

	err = cli.SendData(ctx, TestData{Alpha: "abc345", Beta: -678})
	require.NoError(t, err)

	err = cli.SendData(ctx, nil)
	require.Error(t, err)
}

func TestReceiveData(t *testing.T) {
	t.Parallel()

	type TestData struct {
		Alpha string
		Beta  int
	}

	tests := []struct {
		name    string
		mock    SQS
		data    TestData
		want    string
		wantErr bool
	}{
		{
			name: "success",
			mock: sqsmock{receiveFn: func(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
				return &sqs.ReceiveMessageOutput{
					Messages: []types.Message{
						{
							Body:          aws.String("Kf+BAwEBCFRlc3REYXRhAf+CAAECAQVBbHBoYQEMAAEEQmV0YQEEAAAAD/+CAQZhYmMxMjMB/gLtAA=="),
							ReceiptHandle: aws.String("TestReceiptHandle02"),
						},
					},
				}, nil
			}},
			data:    TestData{Alpha: "abc123", Beta: -375},
			want:    "TestReceiptHandle02",
			wantErr: false,
		},
		{
			name: "empty",
			mock: sqsmock{receiveFn: func(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
				return &sqs.ReceiveMessageOutput{}, nil
			}},
			want:    "",
			wantErr: false,
		},
		{
			name: "error",
			mock: sqsmock{receiveFn: func(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
				return nil, fmt.Errorf("error")
			}},
			want:    "",
			wantErr: true,
		},
		{
			name: "invalid message",
			mock: sqsmock{receiveFn: func(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
				return &sqs.ReceiveMessageOutput{
					Messages: []types.Message{
						{
							Body:          aws.String("你好世界"),
							ReceiptHandle: aws.String("TestReceiptHandle03"),
						},
					},
				}, nil
			}},
			want:    "TestReceiptHandle03",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.TODO()
			cli, err := New(ctx, "https://test_queue.invalid/queue5.fifo", "TEST_MSG_GROUP_ID_5")
			require.NoError(t, err)
			require.NotNil(t, cli)

			cli.sqs = tt.mock

			var data TestData

			got, err := cli.ReceiveData(ctx, &data)
			if tt.wantErr {
				require.Error(t, err)
				require.Equal(t, tt.want, got)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.want, got)
			require.Equal(t, tt.data.Alpha, data.Alpha)
			require.Equal(t, tt.data.Beta, data.Beta)
		})
	}
}

func TestHealthCheck(t *testing.T) {
	t.Parallel()

	queueURL := "https://test_queue.invalid/queue6.fifo"

	tests := []struct {
		name    string
		mock    SQS
		wantErr bool
	}{
		{
			name: "success",
			mock: sqsmock{getQueueAttributesFn: func(ctx context.Context, params *sqs.GetQueueAttributesInput, optFns ...func(*sqs.Options)) (*sqs.GetQueueAttributesOutput, error) {
				return &sqs.GetQueueAttributesOutput{
					Attributes: map[string]string{string(types.QueueAttributeNameLastModifiedTimestamp): "2022-01-02 03:04:05"},
				}, nil
			}},
			wantErr: false,
		},
		{
			name: "no queue",
			mock: sqsmock{getQueueAttributesFn: func(ctx context.Context, params *sqs.GetQueueAttributesInput, optFns ...func(*sqs.Options)) (*sqs.GetQueueAttributesOutput, error) {
				return &sqs.GetQueueAttributesOutput{}, nil
			}},
			wantErr: true,
		},
		{
			name: "error",
			mock: sqsmock{getQueueAttributesFn: func(ctx context.Context, params *sqs.GetQueueAttributesInput, optFns ...func(*sqs.Options)) (*sqs.GetQueueAttributesOutput, error) {
				return &sqs.GetQueueAttributesOutput{}, fmt.Errorf("error")
			}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.TODO()
			cli, err := New(ctx, queueURL, "TEST_MSG_GROUP_ID_6")
			require.NoError(t, err)
			require.NotNil(t, cli)

			cli.sqs = tt.mock

			err = cli.HealthCheck(ctx)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}
